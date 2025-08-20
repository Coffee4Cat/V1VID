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

	fmt.Println("🎥 System kamer WebRTC uruchomiony!")
	fmt.Printf("📡 Główne API dostępne na porcie %d\n", structs.BasePort-1)
	fmt.Println("📡 Wykryte kamery:")
	camera.ListAvailableCameras()
	select {}
}
