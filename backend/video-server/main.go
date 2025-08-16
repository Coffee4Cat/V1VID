package main

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

// Struktura wiadomości sygnalizacyjnej
type SignalingMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// Struktura kamery
type Camera struct {
	ID       string
	Device   string // np. "/dev/video0" lub "0" na Windows
	Track    *webrtc.TrackLocalStaticSample
	FFmpeg   *exec.Cmd
	IsActive bool
	mutex    sync.RWMutex
}

// Manager kamer
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

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // W LAN możemy to uprościć
		},
	}
	cameraManager = &CameraManager{
		cameras: make(map[string]*Camera),
	}
)

func main() {
	// Konfiguracja WebRTC dla LAN
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			// Dla LAN możemy użyć pustej listy lub lokalny STUN
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	}

	// Inicjalizacja kamer (automatyczne wykrywanie lub konfiguracja)
	initializeCameras()

	// WebSocket endpoint
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(w, r, config)
	})

	// API endpoints
	http.HandleFunc("/server-status", serverStatusHandler)
	http.HandleFunc("/api/cameras", handleCamerasAPI)
	http.HandleFunc("/api/camera/start/", handleStartCamera)
	http.HandleFunc("/api/camera/stop/", handleStopCamera)

	// Frontend
	http.Handle("/", http.FileServer(http.Dir("./static/")))

	fmt.Println("🎥 System kamer WebRTC uruchomiony na porcie 8080")
	fmt.Println("📡 Wykryte kamery:")
	listAvailableCameras()

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func initializeCameras() {
	// Automatyczne wykrywanie kamer Linux/Windows
	cameras := detectCameras()

	for i, device := range cameras {
		cameraID := fmt.Sprintf("camera_%d", i)
		camera := &Camera{
			ID:       cameraID,
			Device:   device,
			IsActive: false,
		}

		cameraManager.mutex.Lock()
		cameraManager.cameras[cameraID] = camera
		cameraManager.mutex.Unlock()

		log.Printf("✅ Zarejestrowano kamerę: %s (urządzenie: %s)", cameraID, device)
	}
}

func detectCameras() []string {
	// Prosta detekcja - można rozszerzyć
	var cameras []string

	// Linux
	for i := 0; i < 4; i++ {
		device := fmt.Sprintf("/dev/video%d", i)
		if fileExists(device) {
			cameras = append(cameras, device)
		}
	}

	// Windows (jeśli Linux nie znalazł)
	if len(cameras) == 0 {
		for i := 0; i < 4; i++ {
			cameras = append(cameras, fmt.Sprintf("%d", i))
		}
	}

	return cameras
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

	// Utworzenie H.264 track
	h264Track, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{MimeType: "video/H264"},
		"video",
		camera.ID,
	)
	if err != nil {
		return fmt.Errorf("błąd tworzenia H.264 track: %v", err)
	}

	camera.Track = h264Track

	// Uruchomienie FFmpeg dla H.264
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

			// jeśli ramka była IDR, doklej SPS/PPS na początek
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
			// DIAGNOSTICS HERE!!!
			// else {
			// 	log.Printf("✅ Wysłano ramkę IDR=%v, len=%d", currentIsIDR, len(frameBuffer))
			// }

			// reset na następną ramkę
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
				// inne ignorujemy
			}
		}
	}()

	return nil
}

func buildFFmpegCommand(device string) *exec.Cmd {
	// Konfiguracja FFmpeg dla H.264 z niskim opóźnieniem
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

	// Na Windows zmień -f v4l2 na -f dshow i format urządzenia
	// args[1] = "dshow"
	// args[3] = fmt.Sprintf("video=USB2.0 PC CAMERA") // nazwa urządzenia Windows

	return exec.Command("ffmpeg", args...)
}

func handleWebSocket(w http.ResponseWriter, r *http.Request, config webrtc.Configuration) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("❌ Błąd WebSocket upgrade: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("🔌 Nowe połączenie WebSocket")

	// Utworzenie PeerConnection
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		log.Printf("❌ Błąd PeerConnection: %v", err)
		return
	}
	defer peerConnection.Close()

	// Dodaj wszystkie aktywne kamery do PeerConnection
	cameraManager.mutex.RLock()
	for _, camera := range cameraManager.cameras {
		camera.mutex.RLock()
		if camera.IsActive && camera.Track != nil {
			if _, err := peerConnection.AddTrack(camera.Track); err != nil {
				log.Printf("❌ Błąd dodawania track kamery %s: %v", camera.ID, err)
			} else {
				log.Printf("✅ Dodano track kamery %s do PeerConnection", camera.ID)
			}
		}
		camera.mutex.RUnlock()
	}
	cameraManager.mutex.RUnlock()

	// ICE candidates
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

	// Stan połączenia
	peerConnection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("🔄 Stan WebRTC: %s", state.String())
	})

	// Obsługa wiadomości
	for {
		var msg SignalingMessage
		if err := conn.ReadJSON(&msg); err != nil {
			log.Printf("❌ Błąd WebSocket: %v", err)
			break
		}

		log.Printf("📨 Otrzymano: %s", msg.Type)

		switch msg.Type {
		case "viewer":
			// Klient chce oglądać - wyślij offer
			offer, err := peerConnection.CreateOffer(nil)
			if err != nil {
				log.Printf("❌ Błąd create offer: %v", err)
				continue
			}

			if err := peerConnection.SetLocalDescription(offer); err != nil {
				log.Printf("❌ Błąd set local description: %v", err)
				continue
			}

			offerMsg := SignalingMessage{Type: "offer", Data: offer}
			if err := conn.WriteJSON(offerMsg); err != nil {
				log.Printf("❌ Błąd wysyłania offer: %v", err)
			} else {
				log.Printf("✅ Wysłano offer do viewera")
			}

		case "answer":
			answerData, _ := json.Marshal(msg.Data)
			var answer webrtc.SessionDescription
			json.Unmarshal(answerData, &answer)
			peerConnection.SetRemoteDescription(answer)
			log.Printf("✅ Ustawiono answer")

		case "ice-candidate":
			candidateData, _ := json.Marshal(msg.Data)
			var candidate webrtc.ICECandidateInit
			json.Unmarshal(candidateData, &candidate)
			peerConnection.AddICECandidate(candidate)
		}
	}
}

// API endpoints
func handleCamerasAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")
	cameraManager.mutex.RLock()
	defer cameraManager.mutex.RUnlock()

	type CameraInfo struct {
		ID       string `json:"id"`
		Device   string `json:"device"`
		IsActive bool   `json:"isActive"`
	}

	var cameras []CameraInfo
	for _, camera := range cameraManager.cameras {
		camera.mutex.RLock()
		cameras = append(cameras, CameraInfo{
			ID:       camera.ID,
			Device:   camera.Device,
			IsActive: camera.IsActive,
		})
		camera.mutex.RUnlock()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cameras)
}

func serverStatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	resp := CameraStatusResponse{Status: true, CameraNum: 1} /// Last field unusted
	err := json.NewEncoder(w).Encode(resp)
	if err != nil {
		fmt.Println("sth went wrong")
		return
	}
}

func handleStartCamera(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")
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
	camera.mutex.Lock()
	camera.IsActive = true
	camera.mutex.Unlock()
	resp := CameraStatusResponse{Status: true, CameraNum: 1} /// Last field unusted
	err := json.NewEncoder(w).Encode(resp)
	if err != nil {
		fmt.Println("sth went wrong")
		return
	}
	log.Printf("✅ Uruchomiono kamerę %s", cameraID)
	w.WriteHeader(http.StatusOK)
}

func handleStopCamera(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")
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

	log.Printf("⏹️ Zatrzymano kamerę %s", cameraID)
	w.WriteHeader(http.StatusOK)
}

func listAvailableCameras() {
	cameraManager.mutex.RLock()
	defer cameraManager.mutex.RUnlock()

	if len(cameraManager.cameras) == 0 {
		fmt.Println("❌ Nie znaleziono kamer")
		return
	}

	for _, camera := range cameraManager.cameras {
		fmt.Printf("   📷 %s -> %s\n", camera.ID, camera.Device)
	}
}
