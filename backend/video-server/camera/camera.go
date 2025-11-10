package camera

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"vivid/network"
	"vivid/structs"

	"github.com/pion/webrtc/v3"
)

func detectCameras() []*structs.Camera {
	camera_vec := make([]*structs.Camera, 0)
	cmd := exec.Command("v4l2-ctl", "--list-devices")

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("[ERROR] v4l2-ctl")
	}
	output_string := strings.Split(string(output), "\n")
	camera := &structs.Camera{
		Devs: structs.Devices{EveryDevice: make([]string, 0)},
	}
	for _, line := range output_string {
		if len(line) != 0 {
			if strings.Contains(line, "/dev/video") {
				camera.Devs.EveryDevice = append(camera.Devs.EveryDevice, strings.TrimSpace(line))
			} else if !strings.Contains(line, "/dev/") {
				camera.Name = line
			}
		} else {
			structs.ClasifyCamera(camera)
			switch camera.CamType {
			case structs.MJPEG:
				camera.CurrentDevice = camera.Devs.MJPGDevice
			case structs.H264:
				camera.CurrentDevice = camera.Devs.H264Device
			}
			camera_vec = append(camera_vec, camera)
			camera = &structs.Camera{
				Devs:     structs.Devices{EveryDevice: make([]string, 0)},
				IsActive: false,
			}
		}

	}

	if len(camera_vec) >= 1 {
		camera_vec = camera_vec[:len(camera_vec)-1]
	}

	for _, cam := range camera_vec {
		cam.Represent()
	}
	return camera_vec
}

func createNamedPipe(cameraID string) (string, error) {
	pipeDir := "/tmp/camera_pipes"
	if err := os.MkdirAll(pipeDir, 0755); err != nil {
		return "", fmt.Errorf("couldn't create pipe: %v", err)
	}

	pipePath := filepath.Join(pipeDir, fmt.Sprintf("camera_%s.pipe", cameraID))

	os.Remove(pipePath)

	if err := syscall.Mkfifo(pipePath, 0644); err != nil {
		return "", fmt.Errorf("couldn't create pipe: %v", err)
	}

	return pipePath, nil
}

func InitializeCameras() {
	devices := detectCameras()

	for i, device := range devices {
		cameraID := fmt.Sprintf("camera_%d", i)
		port := structs.BasePort + i
		pipepath, _ := createNamedPipe(cameraID)

		device.ID = cameraID
		device.Port = port
		device.Quality = 4
		device.PipePath = pipepath

		structs.Manager.MMutex.Lock()
		structs.Manager.Cameras[cameraID] = device
		structs.Manager.MMutex.Unlock()

		log.Printf("\033[31m[DEVICE REGISTERED]\033[0m: %s (Name: %s, port: %d, pipepath: %s)\n", cameraID, device.Name, port, pipepath)
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
		fmt.Printf("[DEVICE] %s -> %s (port: %d)\n", camera.ID, camera.CurrentDevice, camera.Port)
	}
}
