package main

import (
	"fmt"
	"vivid/camera"
	"vivid/network"
	"vivid/structs"

	"github.com/go-gst/go-gst/gst"
	"github.com/pion/webrtc/v3"
)

func main() {
	gst.Init(nil)
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{}},
		},
	}

	camera.InitializeCameras()
	camera.StartCameraServers(config)
	network.SetupMainAPIServer()

	fmt.Println("🎥 System kamer WebRTC uruchomiony!")
	fmt.Printf("📡 Główne API dostępne na porcie %d\n", structs.BasePort-1)
	fmt.Println("📡 Wykryte kamery:")
	camera.ListAvailableCameras()
	select {}
}

// import (
// 	"fmt"

// 	"github.com/go-gst/go-glib/glib"
// 	"github.com/go-gst/go-gst/gst"
// 	"github.com/go-gst/go-gst/gst/app"
// )

// func main() {
// 	gst.Init(nil)

// 	// Pipeline jako string
// 	pipelineStr := `
//     v4l2src device=/dev/video4 do-timestamp=true io-mode=2 !
//     video/x-h264,width=1920,height=1080,framerate=30/1 !
//     h264parse config-interval=1 disable-passthrough=true !
//     queue max-size-buffers=1 leaky=downstream !
//     appsink name=mysink emit-signals=true drop=true sync=false
//     `

// 	// Tworzymy pipeline
// 	pipeline, err := gst.NewPipelineFromString(pipelineStr)
// 	if err != nil {
// 		panic(err)
// 	}

// 	// Pobieramy appsink
// 	el, _ := pipeline.GetElementByName("mysink")
// 	if el == nil {
// 		panic("appsink not found")
// 	}

// 	appsink := app.SinkFromElement(el)
// 	if appsink == nil {
// 		panic("element is not an AppSink")
// 	}

// 	// Callback dla nowych próbek
// 	appsink.SetCallbacks(&app.SinkCallbacks{
// 		NewSampleFunc: func(s *app.Sink) gst.FlowReturn {
// 			sample := s.PullSample()
// 			if sample == nil {
// 				return gst.FlowEOS
// 			}

// 			buf := sample.GetBuffer()
// 			pts := buf.PresentationTimestamp()
// 			fmt.Println("Got frame, PTS:", pts)

// 			// Można tu odczytać dane z buf, np.:
// 			// data := buf.Bytes()
// 			return gst.FlowOK
// 		},
// 	})

// 	// Uruchamiamy pipeline
// 	loop := glib.NewMainLoop(nil, false)
// 	pipeline.SetState(gst.StatePlaying)
// 	loop.Run()
// 	pipeline.SetState(gst.StateNull)
// }
