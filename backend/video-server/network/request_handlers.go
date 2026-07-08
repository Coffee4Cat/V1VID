package network

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"vivid/structs"
)

func SetupMainAPIServer() {
	mux := http.NewServeMux()

	mux.HandleFunc("/server-status", ServerStatusHandler)
	mux.HandleFunc("/api/cameras", HandleCamerasAPI)
	mux.HandleFunc("/api/camera/start/", HandleStartCamera)
	mux.HandleFunc("/api/camera/stop/", HandleStopCamera)
	mux.HandleFunc("/api/camera/indorquality/", HandleIndorQualitySpecificCamera)
	mux.HandleFunc("/api/camera/cloudyquality/", HandleCloudyQualitySpecificCamera)
	mux.HandleFunc("/api/camera/sunnyquality/", HandleSunnyQualitySpecificCamera)
	mux.HandleFunc("/api/camera/mjpgquality/", HandleMjpgQualitySpecificCamera)

	mux.Handle("/", http.FileServer(http.Dir("./static/")))

	mainPort := structs.BasePort - 1
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", mainPort),
		Handler: mux,
	}

	go func() {
		log.Printf("Main server on port %d", mainPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[ERROR] Main server failure: %v", err)
		}
	}()
}

func HandleStartSpecificCamera(w http.ResponseWriter, r *http.Request, camera *structs.Camera) {
	structs.SetCORSHeaders(w)

	if err := StartCameraStream(camera); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := structs.CameraStatusResponse{Status: true, CameraNum: 1}
	json.NewEncoder(w).Encode(resp)
	log.Printf("[DEVICE] Initialized Camera: %s", camera.ID)
}

func HandleStopSpecificCamera(w http.ResponseWriter, r *http.Request, camera *structs.Camera) {
	structs.SetCORSHeaders(w)

	camera.MMutex.Lock()
	if camera.FFmpeg != nil {
		camera.FFmpeg.Process.Kill()
		camera.FFmpeg = nil
	}
	camera.IsActive = false
	camera.MMutex.Unlock()

	resp := structs.CameraStatusResponse{Status: false, CameraNum: 1}
	json.NewEncoder(w).Encode(resp)
	log.Printf("[DEVICE] Camera Stopped: %s", camera.ID)
}

func HandleIndorQualitySpecificCamera(w http.ResponseWriter, r *http.Request) {
	structs.SetCORSHeaders(w)
	cameraID := r.URL.Path[len("/api/camera/indorquality/"):]
	structs.Manager.MMutex.RLock()
	camera, _ := structs.Manager.Cameras[cameraID]
	var resp structs.CameraStatusResponse
	if camera.CamType == structs.H264 {
		camera.Quality = 1
		camera.CurrentDevice = camera.Devs.H264Device
		resp = structs.CameraStatusResponse{Status: true, CameraNum: 1}
	} else {
		camera.Quality = 4
		camera.CurrentDevice = camera.Devs.MJPGDevice
		resp = structs.CameraStatusResponse{Status: false, CameraNum: 1}
	}
	json.NewEncoder(w).Encode(resp)
	log.Printf("[MODE] Camera %s in INDOR Mode (device: %s)", camera.ID, camera.CurrentDevice)
	BuildV4L2Command(camera.CurrentDevice, camera.Quality)
	structs.Manager.MMutex.RUnlock()

}

func HandleCloudyQualitySpecificCamera(w http.ResponseWriter, r *http.Request) {
	structs.SetCORSHeaders(w)
	cameraID := r.URL.Path[len("/api/camera/cloudyquality/"):]
	structs.Manager.MMutex.RLock()
	camera, _ := structs.Manager.Cameras[cameraID]
	var resp structs.CameraStatusResponse
	if camera.CamType == structs.H264 {
		camera.Quality = 2
		camera.CurrentDevice = camera.Devs.H264Device
		resp = structs.CameraStatusResponse{Status: true, CameraNum: 1}
	} else {
		camera.Quality = 4
		camera.CurrentDevice = camera.Devs.MJPGDevice
		resp = structs.CameraStatusResponse{Status: false, CameraNum: 1}
	}
	BuildV4L2Command(camera.CurrentDevice, camera.Quality)
	structs.Manager.MMutex.RUnlock()

	json.NewEncoder(w).Encode(resp)
	log.Printf("[MODE] Camera %s in CLOUDY Mode (device: %s)", camera.ID, camera.CurrentDevice)
}

func HandleSunnyQualitySpecificCamera(w http.ResponseWriter, r *http.Request) {
	structs.SetCORSHeaders(w)
	cameraID := r.URL.Path[len("/api/camera/sunnyquality/"):]
	structs.Manager.MMutex.RLock()
	camera, _ := structs.Manager.Cameras[cameraID]
	var resp structs.CameraStatusResponse
	if camera.CamType == structs.H264 {
		camera.Quality = 3
		camera.CurrentDevice = camera.Devs.H264Device
		resp = structs.CameraStatusResponse{Status: true, CameraNum: 1}
	} else {
		camera.Quality = 4
		camera.CurrentDevice = camera.Devs.MJPGDevice
		resp = structs.CameraStatusResponse{Status: false, CameraNum: 1}
	}
	BuildV4L2Command(camera.CurrentDevice, camera.Quality)
	structs.Manager.MMutex.RUnlock()
	json.NewEncoder(w).Encode(resp)
	log.Printf("[MODE] Camera %s in SUNNY Mode (device: %s)", camera.ID, camera.CurrentDevice)
}

func HandleMjpgQualitySpecificCamera(w http.ResponseWriter, r *http.Request) {
	structs.SetCORSHeaders(w)
	cameraID := r.URL.Path[len("/api/camera/mjpgquality/"):]
	structs.Manager.MMutex.RLock()
	camera, _ := structs.Manager.Cameras[cameraID]
	camera.Quality = 4
	camera.CurrentDevice = camera.Devs.MJPGDevice
	BuildV4L2Command(camera.CurrentDevice, camera.Quality)
	structs.Manager.MMutex.RUnlock()

	resp := structs.CameraStatusResponse{Status: true, CameraNum: 1}
	json.NewEncoder(w).Encode(resp)
	log.Printf("[MODE] Camera %s in MJPG Mode (device: %s)", camera.ID, camera.CurrentDevice)
}

func HandleCameraStatus(w http.ResponseWriter, r *http.Request, camera *structs.Camera) {
	structs.SetCORSHeaders(w)

	camera.MMutex.RLock()
	isActive := camera.IsActive
	camera.MMutex.RUnlock()

	resp := structs.CameraStatusResponse{Status: isActive, CameraNum: 1}
	json.NewEncoder(w).Encode(resp)
}

func HandleCamerasAPI(w http.ResponseWriter, r *http.Request) {
	structs.SetCORSHeaders(w)

	structs.Manager.MMutex.RLock()
	defer structs.Manager.MMutex.RUnlock()

	var cameras []structs.CameraListResponse
	for _, camera := range structs.Manager.Cameras {
		camera.MMutex.RLock()
		cameras = append(cameras, structs.CameraListResponse{
			ID:       camera.ID,
			Device:   camera.CurrentDevice,
			Port:     camera.Port,
			IsActive: camera.IsActive,
			Quality:  camera.Quality,
		})
		camera.MMutex.RUnlock()
	}

	json.NewEncoder(w).Encode(cameras)
}

func ServerStatusHandler(w http.ResponseWriter, r *http.Request) {
	structs.SetCORSHeaders(w)

	resp := structs.ServerStatusResponse{Status: true}
	json.NewEncoder(w).Encode(resp)
}

func HandleStartCamera(w http.ResponseWriter, r *http.Request) {
	structs.SetCORSHeaders(w)
	cameraID := r.URL.Path[len("/api/camera/start/"):]

	structs.Manager.MMutex.RLock()
	camera, exists := structs.Manager.Cameras[cameraID]
	structs.Manager.MMutex.RUnlock()

	if !exists {
		http.Error(w, "[ERROR] Camera not detected", http.StatusNotFound)
		return
	}

	if err := StartCameraStream(camera); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := structs.CameraStatusResponse{Status: true, CameraNum: 1}
	json.NewEncoder(w).Encode(resp)
	log.Printf("[DEVICE] Initialized Camera: %s", cameraID)
}

func HandleStopCamera(w http.ResponseWriter, r *http.Request) {
	structs.SetCORSHeaders(w)
	cameraID := r.URL.Path[len("/api/camera/stop/"):]

	structs.Manager.MMutex.RLock()
	camera, exists := structs.Manager.Cameras[cameraID]
	structs.Manager.MMutex.RUnlock()

	if !exists {
		http.Error(w, "[ERROR] Camera not detected", http.StatusNotFound)
		return
	}

	camera.MMutex.Lock()
	if camera.FFmpeg != nil {
		camera.FFmpeg.Process.Kill()
		camera.FFmpeg = nil
	}
	camera.IsActive = false
	camera.MMutex.Unlock()

	resp := structs.CameraStatusResponse{Status: false, CameraNum: 1}
	json.NewEncoder(w).Encode(resp)
	log.Printf("[DEVICE] Camera Stpped: %s", cameraID)
}
