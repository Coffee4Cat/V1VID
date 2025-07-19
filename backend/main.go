package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
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

	cmd := exec.Command("ffmpeg",
		// Wejście z kamery
		"-f", "v4l2",
		"-input_format", "mjpeg",
		"-framerate", "30",
		"-video_size", "1280x720",
		"-i", "/dev/video0",

		// Kodek i profil niskiej latencji
		"-vcodec", "libx264",
		"-preset", "ultrafast",
		"-tune", "zerolatency",
		"-g", "30",
		"-keyint_min", "30",
		"-sc_threshold", "0",
		"-bf", "0",

		// Format kontenera i adres SRT (listener)
		"-f", "mpegts",
		"srt://0.0.0.0:8888?mode=listener&latency=50&pkt_size=1316",
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start ffmpeg: %v", err)
	}

	log.Println("ffmpeg started, streaming via SRT on port 8888...")

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	<-c

	log.Println("Interrupt received, stopping ffmpeg...")

	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		log.Printf("Failed to send interrupt to ffmpeg: %v", err)
	}

	cmd.Wait()
	log.Println("ffmpeg stopped, exiting")

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
