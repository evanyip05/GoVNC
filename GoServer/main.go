package main

import (
	"Tools/RTC"
	"bytes"
	"encoding/base64"
	//"image/jpeg"
	"image/png"
	"time"

	"github.com/kbinani/screenshot"
	"github.com/nfnt/resize"
	"github.com/pion/webrtc/v3"
	//"github.com/go-vgo/robotgo"
)

func chunk(s string, chunkSize int) []string {
	var chunks []string

	for len(s) > 0 {
		if len(s) <= chunkSize {
			chunks = append(chunks, s)
			break
		}

		chunks = append(chunks, s[:chunkSize])
		s = s[chunkSize:]
	}

	return chunks
}

func DoRTC() {
	client, err := RTC.NewClient("52.14.127.124:8080", "stun:stun1.l.google.com:19302", "stun:stun2.l.google.com:19302")
	if err != nil {println("Main err 1:", err.Error()); return}

	client.Connect("AA")

	dc, err := client.SendDataChannel("test")
	if err != nil {println("Main err 2:", err.Error()); return}

	ch := make(chan bool)

	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		ch <- true
	})

	last := ""
	//sx, sy := robotgo.GetScreenSize()
	//println(sx, sy)

	//delay := 16 * time.Millisecond

	

	for {
		
		img, err := screenshot.Capture(0,0,1920,1080)
		if err != nil {println("main cap err 1:", img); continue}


		
		buf := new(bytes.Buffer)
		err = png.Encode(buf, resize.Resize(640, 360, img, resize.MitchellNetravali))//, &jpeg.Options{Quality: 50})
		if err != nil {println("main cap err 2:", img); continue}
		
	

		b64 := base64.StdEncoding.EncodeToString(buf.Bytes())

		//println(len(b64))

		if (b64 != last) {
			chunks := chunk(b64, 1000)

			for _, c := range chunks {
				dc.SendText(c)
			}
			dc.SendText("END_OF_FRAME")

			last = b64
		}

		select {
			case <- ch: 
			case <- time.After(100*time.Millisecond):
				println("timed out")
		}

		//println("sending iamge....")
		

		

		//time.Sleep(delay)
	}
}

func main() {
	DoRTC()



	select{}
}