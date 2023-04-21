package main

import (
	"crypto/md5"
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type imageBlock struct {
	i     *image.RGBA
	isNew bool
}

func main() {
	fn := flag.String("device", "/dev/fb0", "Framebuffer device to read")
	port := flag.Int64("port", 9876, "Port to service mjpeg stream")
	flag.Parse()
	file, err := os.Open(*fn)
	if err != nil {
		log.Fatalf("Couldn't open console device: %s\n", err.Error())
	}
	file.Close()

	c := make(chan imageBlock, 1)
	go fbLoop(*fn, c)
	o := &jpeg.Options{Quality: 75}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "multipart/x-mixed-replace; boundary=frame")
		boundary := "\r\n--frame\r\nContent-Type: image/jpeg\r\n\r\n"
		for {
			i := <-c
			if i.isNew {
				n, err := io.WriteString(w, boundary)
				if err != nil || n != len(boundary) {
					return
				}

				err = jpeg.Encode(w, i.i, o)
				if err != nil {
					return
				}

				n, err = io.WriteString(w, "\r\n")
				if err != nil || n != 2 {
					return
				}
			}
			time.Sleep(300 * time.Millisecond)

		}
	})
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}

func fbLoop(file string, c chan imageBlock) {
	res := 640 * 480
	t := make([]byte, res*3)
	rgba := make([]byte, res*4)
	var i imageBlock
	i.i = image.NewRGBA(image.Rect(0, 0, 640, 480))
	var l [16]byte
	for {
		f, err := os.Open(file)
		if err != nil {
			log.Fatalf("Error opening console device: %s", err.Error())
		}
		n, err := io.ReadFull(f, t)
		f.Close()
		h := md5.Sum(t)
		if h == l {
			i.isNew = false
			c <- i
			continue
		}
		i.isNew = true
		l = h
		if err != nil {
			log.Fatalf("Read error: %s", err.Error())
		}
		if n != res*3 {
			log.Fatal("Short read")
		}
		for x := 0; x < (640 * 480); x++ {
			rgba[x*4] = t[x*3]
			rgba[(x*4)+1] = t[(x*3)+1]
			rgba[(x*4)+2] = t[(x*3)+2]
			rgba[(x*4)+3] = 255
		}
		i.i.Pix = rgba
		c <- i
	}

}
