package main

import (
	"fmt"
	"vivid/camera"
	"vivid/network"
	"vivid/structs"

	"github.com/pion/webrtc/v3"
)

func main() {
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{}},
		},
	}

	camera.InitializeCameras()
	camera.StartCameraServers(config)
	network.SetupMainAPIServer()

	fmt.Println("++V1VID++")
	fmt.Printf("MAIN api on port %d\n", structs.BasePort-1)
	fmt.Println("[DEVICE] Detected Cameras:")
	camera.ListAvailableCameras()
	select {}
}
