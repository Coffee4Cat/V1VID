package network

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"time"
	"vivid/structs"

	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/pion/webrtc/v3/pkg/media/h264reader"
)

func BuildFFmpegCommand(device string) *exec.Cmd {
	args := []string{
		"-f", "v4l2",
		"-input_format", "mjpeg", // dodane - czytanie w MJPEG
		"-framerate", "30",
		"-video_size", "1024x576",
		"-i", device,
		"-c:v", "libx264",
		"-preset", "fast",
		"-profile:v", "baseline",
		"-tune", "zerolatency",
		"-pix_fmt", "yuv420p",
		"-r", "30",
		"-b:v", "4M",
		"-maxrate", "5M",
		"-bufsize", "8M",
		"-g", "30",
		"-x264opts", "keyint=30:no-scenecut:aud",
		"-fflags", "nobuffer",
		"-flags", "low_delay",
		"-f", "h264",
		"-",
	}

	// WOLNO
	// args := []string{
	// 	"-f", "v4l2",
	// 	"-input_format", "h264",
	// 	"-framerate", "30",
	// 	"-video_size", "1024x576",
	// 	"-i", device,
	// 	"-c:v", "libx264", // lekkie rekodowanie
	// 	"-preset", "ultrafast", // najszybszy preset
	// 	"-tune", "zerolatency",
	// 	"-profile:v", "baseline", // profil zgodny z WebRTC
	// 	"-pix_fmt", "yuv420p",
	// 	"-g", "30", // keyframe co sekundę
	// 	"-x264opts", "keyint=30:no-scenecut:aud", // NAL AUD units dla WebRTC
	// 	"-f", "h264",
	// 	"-fflags", "nobuffer",
	// 	"-flags", "low_delay",
	// 	"-",
	// }

	return exec.Command("ffmpeg", args...)

}

func HandleCameraWebSocket(w http.ResponseWriter, r *http.Request, camera *structs.Camera, config webrtc.Configuration) {
	conn, err := structs.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("❌ Błąd WebSocket upgrade dla kamery %s: %v", camera.ID, err)
		return
	}

	safeConn := structs.NewSafeWebSocketConn(conn)
	defer safeConn.Close()

	log.Printf("🔌 Nowe połączenie WebSocket dla kamery %s", camera.ID)

	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		log.Printf("❌ Błąd PeerConnection dla kamery %s: %v", camera.ID, err)
		return
	}
	defer peerConnection.Close()

	camera.MMutex.RLock()
	if camera.IsActive && camera.Track != nil {
		if _, err := peerConnection.AddTrack(camera.Track); err != nil {
			log.Printf("❌ Błąd dodawania track kamery %s: %v", camera.ID, err)
		} else {
			log.Printf("✅ Dodano track kamery %s do PeerConnection", camera.ID)
		}
	}
	camera.MMutex.RUnlock()

	// TUTAJ JEST POPRAWKA - używamy SafeWebSocketConn zamiast bezpośredniego conn.WriteJSON
	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}
		candidateMsg := structs.SignalingMessage{
			Type: "ice-candidate",
			Data: candidate.ToJSON(),
		}
		// Bezpieczne wysyłanie przez kanał
		safeConn.SendMessage(candidateMsg)
	})

	peerConnection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("🔄 Stan WebRTC dla kamery %s: %s", camera.ID, state.String())
	})

	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var msg structs.SignalingMessage
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
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

			offerMsg := structs.SignalingMessage{Type: "offer", Data: offer}
			// POPRAWKA - używamy safeConn zamiast conn.WriteJSON
			if err := safeConn.SendMessage(offerMsg); err != nil {
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
func StartCameraStream(camera *structs.Camera) error {
	camera.MMutex.Lock()
	defer camera.MMutex.Unlock()

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

	ffmpegCmd := BuildFFmpegCommand(camera.Device)

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
			camera.MMutex.Lock()
			camera.IsActive = false
			camera.MMutex.Unlock()

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
