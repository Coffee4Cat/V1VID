package camera

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"vivid/network"
	"vivid/structs"

	"github.com/pion/webrtc/v3"
)

func detectCameras() []string {
	var cameras []string

	// search hardcoded for now - needs automation
	for i := 0; i < 25; i++ {
		// if i == 0 || i == 2 || i == 6 || i == 10 || i == 14 || i == 18 || i == 22 {
		if i == 2 || i == 6 || i == 10 || i == 14 || i == 18 {
			device := fmt.Sprintf("/dev/video%d", i)
			if exec.Command("ls", device).Run() == nil {
				cameras = append(cameras, device)
			}
		}
	}

	return cameras
}

func createNamedPipe(cameraID string) (string, error) {
	pipeDir := "/tmp/camera_pipes"
	if err := os.MkdirAll(pipeDir, 0755); err != nil {
		return "", fmt.Errorf("Couldn't create pipe: %v", err)
	}

	pipePath := filepath.Join(pipeDir, fmt.Sprintf("camera_%s.pipe", cameraID))

	os.Remove(pipePath)

	if err := syscall.Mkfifo(pipePath, 0644); err != nil {
		return "", fmt.Errorf("Could'nt create pipe: %v", err)
	}

	return pipePath, nil
}

func InitializeCameras() {
	devices := detectCameras()

	for i, device := range devices {
		cameraID := fmt.Sprintf("camera_%d", i)
		port := structs.BasePort + i
		pipepath, _ := createNamedPipe(cameraID)

		camera := &structs.Camera{
			ID:       cameraID,
			Device:   device,
			Port:     port,
			IsActive: false,
			Quality:  1,
			PipePath: pipepath,
		}

		structs.Manager.MMutex.Lock()
		structs.Manager.Cameras[cameraID] = camera
		structs.Manager.MMutex.Unlock()

		log.Printf("[DEVICE REGISTERED]: %s (dev: %s, port: %d, pipepath: %s)", cameraID, device, port, pipepath)
	}
}

func startCameraServer(camera *structs.Camera, config webrtc.Configuration) {
	mux := http.NewServeMux()

	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		network.HandleCameraWebSocket(w, r, camera, config)
	})

	mux.HandleFunc("/api/start", func(w http.ResponseWriter, r *http.Request) {
		network.HandleStartSpecificCamera(w, r, camera)
	})

	mux.HandleFunc("/api/stop", func(w http.ResponseWriter, r *http.Request) {
		network.HandleStopSpecificCamera(w, r, camera)
	})

	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		network.HandleCameraStatus(w, r, camera)
	})

	mux.HandleFunc("/api/indorquality", func(w http.ResponseWriter, r *http.Request) {
		network.HandleIndorQualitySpecificCamera(w, r)
	})

	mux.HandleFunc("/api/cloudyquality", func(w http.ResponseWriter, r *http.Request) {
		network.HandleCloudyQualitySpecificCamera(w, r)
	})

	mux.HandleFunc("/api/sunnyquality", func(w http.ResponseWriter, r *http.Request) {
		network.HandleSunnyQualitySpecificCamera(w, r)
	})

	mux.Handle("/", http.FileServer(http.Dir("./static/")))

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", camera.Port),
		Handler: mux,
	}

	camera.MMutex.Lock()
	camera.Server = server
	camera.MMutex.Unlock()

	log.Printf("[LAUNCH] Server for camera %s on port %d", camera.ID, camera.Port)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("[ERROR] %s: %v", camera.ID, err)
	}
}
func StartCameraServers(config webrtc.Configuration) {
	structs.Manager.MMutex.RLock()
	defer structs.Manager.MMutex.RUnlock()

	for _, camera := range structs.Manager.Cameras {
		go startCameraServer(camera, config)
	}
}

func ListAvailableCameras() {
	structs.Manager.MMutex.RLock()
	defer structs.Manager.MMutex.RUnlock()

	if len(structs.Manager.Cameras) == 0 {
		fmt.Println("[ERROR] Cameras not detected")
		return
	}

	for _, camera := range structs.Manager.Cameras {
		fmt.Printf("[DEVICE] %s -> %s (port: %d)\n", camera.ID, camera.Device, camera.Port)
	}
}
