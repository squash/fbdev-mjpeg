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
	width:=flag.Int("width", 640, "Framebuffer width in pixels")
	height:=flag.Int("height", 480, "Framebuffer height in pixels")
	flag.Parse()

	// make sure we can open the framebuffer device before we begin
	file, err := os.Open(*fn)
	if err != nil {
		log.Fatalf("Couldn't open console device: %s\n", err.Error())
	}
	file.Close()

	c := make(chan imageBlock, 1)
	go fbLoop(*fn, *width, *height, c)
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

func fbLoop(file string, width, height int, c chan imageBlock) {
	pixels:=width*height
	t := make([]byte, pixels*3) // 3 bytes per pixel RGB
	rgba := make([]byte, pixels*4) // 4 bytes per pixel RGBA
	var i imageBlock

	i.i = image.NewRGBA(image.Rect(0, 0, width, height))
	// Smart! Pre-fill the Alpha channel bit
	for x:=0;x<pixels;x++ {
		i.i.Pix[(x*4)+3]=255
	}
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
		if n != pixels*3 {
			log.Fatal("Short read")
		}
		for x := 0; x < pixels; x++ {
			rgba[x*4] = t[x*3]
			rgba[(x*4)+1] = t[(x*3)+1]
			rgba[(x*4)+2] = t[(x*3)+2]
		}
		i.i.Pix = rgba
		c <- i
	}

}
