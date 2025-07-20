package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"time"

	"github.com/pion/webrtc/v3"
)

type ServerStatusResponse struct {
	Status bool `json:"status"`
}
type CameraStatusResponse struct {
	Status    bool `json:"status"`
	CameraNum int  `json:"camera_num"`
}

func serverStatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	resp := CameraStatusResponse{Status: true, CameraNum: 1}
	err := json.NewEncoder(w).Encode(resp)
	if err != nil {
		fmt.Println("sth went wrong")
		return
	}
}

func cameraStatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	resp := CameraStatusResponse{Status: true, CameraNum: 1}
	err := json.NewEncoder(w).Encode(resp)
	if err != nil {
		fmt.Println("sth went wrong")
		return
	}
}

func cameraStreamHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	resp := CameraStatusResponse{Status: true, CameraNum: 1}
	err := json.NewEncoder(w).Encode(resp)
	if err != nil {
		fmt.Println("sth went wrong")
		return
	}

	// Odczytujemy SDP offer z klienta
	var offer webrtc.SessionDescription
	if err := json.NewDecoder(r.Body).Decode(&offer); err != nil {
		http.Error(w, "invalid SDP", http.StatusBadRequest)
		return
	}

	// Konfiguracja peer connection
	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Dodajemy track video VP8
	videoTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{
		MimeType: webrtc.MimeTypeVP8,
	}, "video", "pion")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, err = peerConnection.AddTrack(videoTrack)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Ustawiamy handler ICE candidate (w praktyce wyślij klientowi)
	peerConnection.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		fmt.Printf("New ICE candidate: %s\n", c.ToJSON().Candidate)
	})

	if err := peerConnection.SetRemoteDescription(offer); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := peerConnection.SetLocalDescription(answer); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ffmpegCmd := exec.Command("ffmpeg",
		"-f", "v4l2", // Linux kamera, np. /dev/video0
		"-i", "/dev/video0",
		"-c:v", "libvpx", // VP8 encoder
		"-deadline", "realtime",
		"-f", "ivf",
		"pipe:1",
	)

	stdout, err := ffmpegCmd.StdoutPipe()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := ffmpegCmd.Start(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// IVF ma nagłówek 32 bajty i potem co klatka 12 bajtów header + dane
	// Tu upraszczamy i odczytujemy cały strumień do wysłania sample po sample do tracka

	go func() {
		defer ffmpegCmd.Process.Kill()
		// Pomijamy nagłówek IVF (32 bajty)
		buf := make([]byte, 32)
		if _, err := io.ReadFull(stdout, buf); err != nil {
			return
		}

		for {
			// IVF frame header 12 bajtów
			header := make([]byte, 12)
			if _, err := io.ReadFull(stdout, header); err != nil {
				return
			}

			// Wielkość frame to 4 pierwsze bajty (little endian)
			frameSize := int(uint32(header[0]) | uint32(header[1])<<8 | uint32(header[2])<<16 | uint32(header[3])<<24)
			frameData := make([]byte, frameSize)
			if _, err := io.ReadFull(stdout, frameData); err != nil {
				return
			}

			// Wysyłamy sample do tracka (opóźnienie ~33ms)
			err := videoTrack.WriteSample(webrtc.Sample{Data: frameData, Duration: time.Millisecond * 33})
			if err != nil {
				return
			}
		}
	}()

}

func main() {
	http.HandleFunc("/server-status", serverStatusHandler)
	http.HandleFunc("/camera-status", cameraStatusHandler)
	http.HandleFunc("/stream", cameraStreamHandler)

	fmt.Println("Serwer nasłuchuje na 8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println("Błąd uruchomienia serwera: ", err)
	}
}
