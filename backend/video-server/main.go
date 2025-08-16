package main

// well I need to do some refactorization here. Its unreadable at this point
import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/pion/webrtc/v3/pkg/media/h264reader"
)

type SignalingMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type Camera struct {
	ID       string
	Device   string
	Port     int
	Track    *webrtc.TrackLocalStaticSample
	FFmpeg   *exec.Cmd
	IsActive bool
	Server   *http.Server
	mutex    sync.RWMutex
}

type CameraManager struct {
	cameras map[string]*Camera
	mutex   sync.RWMutex
}

type ServerStatusResponse struct {
	Status bool `json:"status"`
}

type CameraStatusResponse struct {
	Status    bool `json:"status"`
	CameraNum int  `json:"camera_num"`
}

type CameraListResponse struct {
	ID       string `json:"id"`
	Device   string `json:"device"`
	Port     int    `json:"port"`
	IsActive bool   `json:"isActive"`
}

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	cameraManager = &CameraManager{
		cameras: make(map[string]*Camera),
	}
	basePort = 8081
)

func main() {
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	}

	initializeCameras()
	startCameraServers(config)
	setupMainAPIServer()

	fmt.Println("🎥 System kamer WebRTC uruchomiony!")
	fmt.Printf("📡 Główne API dostępne na porcie %d\n", basePort-1)
	fmt.Println("📡 Wykryte kamery:")
	listAvailableCameras()
	select {}
}

func initializeCameras() {
	devices := detectCameras()

	for i, device := range devices {
		cameraID := fmt.Sprintf("camera_%d", i)
		port := basePort + i

		camera := &Camera{
			ID:       cameraID,
			Device:   device,
			Port:     port,
			IsActive: false,
		}

		cameraManager.mutex.Lock()
		cameraManager.cameras[cameraID] = camera
		cameraManager.mutex.Unlock()

		log.Printf("✅ Zarejestrowano kamerę: %s (urządzenie: %s, port: %d)", cameraID, device, port)
	}
}

func detectCameras() []string {
	var cameras []string

	for i := 0; i < 4; i++ {
		device := fmt.Sprintf("/dev/video%d", i)
		if fileExists(device) {
			cameras = append(cameras, device)
		}
	}

	return cameras
}

func startCameraServers(config webrtc.Configuration) {
	cameraManager.mutex.RLock()
	defer cameraManager.mutex.RUnlock()

	for _, camera := range cameraManager.cameras {
		go startCameraServer(camera, config)
	}
}

func startCameraServer(camera *Camera, config webrtc.Configuration) {
	mux := http.NewServeMux()

	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handleCameraWebSocket(w, r, camera, config)
	})

	mux.HandleFunc("/api/start", func(w http.ResponseWriter, r *http.Request) {
		handleStartSpecificCamera(w, r, camera)
	})

	mux.HandleFunc("/api/stop", func(w http.ResponseWriter, r *http.Request) {
		handleStopSpecificCamera(w, r, camera)
	})

	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		handleCameraStatus(w, r, camera)
	})

	mux.Handle("/", http.FileServer(http.Dir("./static/")))

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", camera.Port),
		Handler: mux,
	}

	camera.mutex.Lock()
	camera.Server = server
	camera.mutex.Unlock()

	log.Printf("🚀 Uruchamiam serwer dla kamery %s na porcie %d", camera.ID, camera.Port)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("❌ Błąd serwera kamery %s: %v", camera.ID, err)
	}
}

func setupMainAPIServer() {
	mux := http.NewServeMux()

	mux.HandleFunc("/server-status", serverStatusHandler)
	mux.HandleFunc("/api/cameras", handleCamerasAPI)
	mux.HandleFunc("/api/camera/start/", handleStartCamera)
	mux.HandleFunc("/api/camera/stop/", handleStopCamera)

	mux.Handle("/", http.FileServer(http.Dir("./static/")))

	mainPort := basePort - 1
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", mainPort),
		Handler: mux,
	}

	go func() {
		log.Printf("🌐 Uruchamiam główny serwer API na porcie %d", mainPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("❌ Błąd głównego serwera: %v", err)
		}
	}()
}

func handleCameraWebSocket(w http.ResponseWriter, r *http.Request, camera *Camera, config webrtc.Configuration) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("❌ Błąd WebSocket upgrade dla kamery %s: %v", camera.ID, err)
		return
	}
	defer conn.Close()

	log.Printf("🔌 Nowe połączenie WebSocket dla kamery %s", camera.ID)

	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		log.Printf("❌ Błąd PeerConnection dla kamery %s: %v", camera.ID, err)
		return
	}
	defer peerConnection.Close()

	camera.mutex.RLock()
	if camera.IsActive && camera.Track != nil {
		if _, err := peerConnection.AddTrack(camera.Track); err != nil {
			log.Printf("❌ Błąd dodawania track kamery %s: %v", camera.ID, err)
		} else {
			log.Printf("✅ Dodano track kamery %s do PeerConnection", camera.ID)
		}
	}
	camera.mutex.RUnlock()

	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}
		candidateMsg := SignalingMessage{
			Type: "ice-candidate",
			Data: candidate.ToJSON(),
		}
		conn.WriteJSON(candidateMsg)
	})

	peerConnection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("🔄 Stan WebRTC dla kamery %s: %s", camera.ID, state.String())
	})

	for {
		var msg SignalingMessage
		if err := conn.ReadJSON(&msg); err != nil {
			log.Printf("❌ Błąd WebSocket dla kamery %s: %v", camera.ID, err)
			break
		}

		log.Printf("📨 Otrzymano dla kamery %s: %s", camera.ID, msg.Type)

		switch msg.Type {
		case "viewer":
			offer, err := peerConnection.CreateOffer(nil)
			if err != nil {
				log.Printf("❌ Błąd create offer dla kamery %s: %v", camera.ID, err)
				continue
			}

			if err := peerConnection.SetLocalDescription(offer); err != nil {
				log.Printf("❌ Błąd set local description dla kamery %s: %v", camera.ID, err)
				continue
			}

			offerMsg := SignalingMessage{Type: "offer", Data: offer}
			if err := conn.WriteJSON(offerMsg); err != nil {
				log.Printf("❌ Błąd wysyłania offer dla kamery %s: %v", camera.ID, err)
			} else {
				log.Printf("✅ Wysłano offer do viewera dla kamery %s", camera.ID)
			}

		case "answer":
			answerData, _ := json.Marshal(msg.Data)
			var answer webrtc.SessionDescription
			json.Unmarshal(answerData, &answer)
			peerConnection.SetRemoteDescription(answer)
			log.Printf("✅ Ustawiono answer dla kamery %s", camera.ID)

		case "ice-candidate":
			candidateData, _ := json.Marshal(msg.Data)
			var candidate webrtc.ICECandidateInit
			json.Unmarshal(candidateData, &candidate)
			peerConnection.AddICECandidate(candidate)
		}
	}
}

func addStartCode(nal []byte) []byte {
	return append([]byte{0x00, 0x00, 0x00, 0x01}, nal...)
}

func fileExists(filename string) bool {
	cmd := exec.Command("ls", filename)
	return cmd.Run() == nil
}

func startCameraStream(camera *Camera) error {
	camera.mutex.Lock()
	defer camera.mutex.Unlock()

	if camera.IsActive {
		return fmt.Errorf("kamera %s już jest aktywna", camera.ID)
	}

	h264Track, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{MimeType: "video/H264"},
		"video",
		camera.ID,
	)
	if err != nil {
		return fmt.Errorf("błąd tworzenia H.264 track: %v", err)
	}

	camera.Track = h264Track

	ffmpegCmd := buildFFmpegCommand(camera.Device)

	log.Printf("🚀 Uruchamiam FFmpeg dla kamery %s: %s", camera.ID, ffmpegCmd.String())

	stdout, err := ffmpegCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("błąd stdout pipe: %v", err)
	}

	if err := ffmpegCmd.Start(); err != nil {
		return fmt.Errorf("błąd uruchomienia FFmpeg: %v", err)
	}

	camera.FFmpeg = ffmpegCmd
	camera.IsActive = true

	go func() {
		defer func() {
			camera.mutex.Lock()
			camera.IsActive = false
			camera.mutex.Unlock()

			if camera.FFmpeg != nil {
				camera.FFmpeg.Process.Kill()
			}
		}()

		h264Reader, err := h264reader.NewReader(stdout)
		if err != nil {
			log.Printf("❌ Błąd H.264 reader dla kamery %s: %v", camera.ID, err)
			return
		}

		log.Printf("📹 Rozpoczynam streaming H.264 dla kamery %s", camera.ID)

		var sps []byte
		var pps []byte
		var frameBuffer []byte
		var currentIsIDR bool

		flushFrame := func() {
			if len(frameBuffer) == 0 {
				return
			}

			if currentIsIDR && sps != nil && pps != nil {
				out := []byte{}
				out = append(out, addStartCode(sps)...)
				out = append(out, addStartCode(pps)...)
				out = append(out, frameBuffer...)
				frameBuffer = out
			}

			sample := media.Sample{
				Data:     frameBuffer,
				Duration: time.Second / 30,
			}
			if err := h264Track.WriteSample(sample); err != nil {
				if err == io.ErrClosedPipe {
					log.Printf("📹 Track zamknięty dla kamery %s", camera.ID)
					return
				}
				log.Printf("❌ Błąd wysyłania sample: %v", err)
			}

			frameBuffer = nil
			currentIsIDR = false
		}

		for {
			nal, h264Err := h264Reader.NextNAL()
			if h264Err != nil {
				if h264Err == io.EOF {
					log.Printf("📹 Koniec streamu dla kamery %s", camera.ID)
				} else {
					log.Printf("❌ Błąd odczytu H.264: %v", h264Err)
				}
				break
			}

			switch nal.UnitType {
			case 7: // SPS
				sps = nal.Data
			case 8: // PPS
				pps = nal.Data
			case 5: // IDR
				currentIsIDR = true
				frameBuffer = append(frameBuffer, addStartCode(nal.Data)...)
			case 1: // non-IDR slice
				frameBuffer = append(frameBuffer, addStartCode(nal.Data)...)
			case 9: // AUD -> koniec ramki
				flushFrame()
			default:

			}
		}
	}()

	return nil
}

func buildFFmpegCommand(device string) *exec.Cmd {
	// Some change will be required as it does not support dynamic parameter change
	args := []string{
		"-f", "v4l2",
		"-framerate", "30",
		"-video_size", "640x480",
		"-i", device,
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-profile:v", "baseline",
		"-tune", "zerolatency",
		"-pix_fmt", "yuv420p",
		"-r", "30",
		"-b:v", "1M",
		"-maxrate", "1M",
		"-bufsize", "2M",
		"-g", "30",
		"-x264opts", "keyint=30:no-scenecut:aud",
		"-fflags", "nobuffer",
		"-flags", "low_delay",
		"-f", "h264",
		"-",
	}

	return exec.Command("ffmpeg", args...)
}

func handleStartSpecificCamera(w http.ResponseWriter, r *http.Request, camera *Camera) {
	setCORSHeaders(w)

	if err := startCameraStream(camera); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := CameraStatusResponse{Status: true, CameraNum: 1}
	json.NewEncoder(w).Encode(resp)
	log.Printf("✅ Uruchomiono kamerę %s", camera.ID)
}

func handleStopSpecificCamera(w http.ResponseWriter, r *http.Request, camera *Camera) {
	setCORSHeaders(w)

	camera.mutex.Lock()
	if camera.FFmpeg != nil {
		camera.FFmpeg.Process.Kill()
		camera.FFmpeg = nil
	}
	camera.IsActive = false
	camera.mutex.Unlock()

	resp := CameraStatusResponse{Status: false, CameraNum: 1}
	json.NewEncoder(w).Encode(resp)
	log.Printf("⏹️ Zatrzymano kamerę %s", camera.ID)
}

func handleCameraStatus(w http.ResponseWriter, r *http.Request, camera *Camera) {
	setCORSHeaders(w)

	camera.mutex.RLock()
	isActive := camera.IsActive
	camera.mutex.RUnlock()

	resp := CameraStatusResponse{Status: isActive, CameraNum: 1}
	json.NewEncoder(w).Encode(resp)
}

func handleCamerasAPI(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)

	cameraManager.mutex.RLock()
	defer cameraManager.mutex.RUnlock()

	var cameras []CameraListResponse
	for _, camera := range cameraManager.cameras {
		camera.mutex.RLock()
		cameras = append(cameras, CameraListResponse{
			ID:       camera.ID,
			Device:   camera.Device,
			Port:     camera.Port,
			IsActive: camera.IsActive,
		})
		camera.mutex.RUnlock()
	}

	json.NewEncoder(w).Encode(cameras)
}

func serverStatusHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)

	resp := ServerStatusResponse{Status: true}
	json.NewEncoder(w).Encode(resp)
}

func handleStartCamera(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	cameraID := r.URL.Path[len("/api/camera/start/"):]

	cameraManager.mutex.RLock()
	camera, exists := cameraManager.cameras[cameraID]
	cameraManager.mutex.RUnlock()

	if !exists {
		http.Error(w, "Kamera nie znaleziona", http.StatusNotFound)
		return
	}

	if err := startCameraStream(camera); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := CameraStatusResponse{Status: true, CameraNum: 1}
	json.NewEncoder(w).Encode(resp)
	log.Printf("✅ Uruchomiono kamerę %s", cameraID)
}

func handleStopCamera(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	cameraID := r.URL.Path[len("/api/camera/stop/"):]

	cameraManager.mutex.RLock()
	camera, exists := cameraManager.cameras[cameraID]
	cameraManager.mutex.RUnlock()

	if !exists {
		http.Error(w, "Kamera nie znaleziona", http.StatusNotFound)
		return
	}

	camera.mutex.Lock()
	if camera.FFmpeg != nil {
		camera.FFmpeg.Process.Kill()
		camera.FFmpeg = nil
	}
	camera.IsActive = false
	camera.mutex.Unlock()

	resp := CameraStatusResponse{Status: false, CameraNum: 1}
	json.NewEncoder(w).Encode(resp)
	log.Printf("⏹️ Zatrzymano kamerę %s", cameraID)
}

func setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")
}

func listAvailableCameras() {
	cameraManager.mutex.RLock()
	defer cameraManager.mutex.RUnlock()

	if len(cameraManager.cameras) == 0 {
		fmt.Println("❌ Nie znaleziono kamer")
		return
	}

	for _, camera := range cameraManager.cameras {
		fmt.Printf("   📷 %s -> %s (port: %d)\n", camera.ID, camera.Device, camera.Port)
	}
}
