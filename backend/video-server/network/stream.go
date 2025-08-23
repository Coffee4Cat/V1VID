package network

import (
	"bytes"
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
)

func BuildFFmpegCommand(device string, mode int) *exec.Cmd {
	var args []string
	if mode == 1 {

		args = []string{
			"-f", "v4l2",
			"-input_format", "mjpeg",
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
	} else {
		args = []string{
			"-f", "v4l2",
			"-input_format", "mjpeg",
			"-framerate", "30",
			"-video_size", "640x360",
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
	}

	// DOBRE
	// args = []string{
	// 	"v4l2src", "device=/dev/video4", "do-timestamp=true", "num-buffers=-1", "io-mode=2",
	// 	"!", "video/x-h264,width=1280,height=720,framerate=30/1",
	// 	"!", "h264parse", "config-interval=1", "disable-passthrough=true",
	// 	"!", "queue", "max-size-buffers=1", "leaky=downstream",
	// 	"!", "filesink", "location=/dev/stdout", "sync=false", "async=true", "buffer-mode=0",
	// }

	args = []string{
		"v4l2src", "device=/dev/video4",
		"do-timestamp=true",
		"io-mode=2",
		"!", "video/x-h264,width=1280,height=720,framerate=30/1",
		"!", "h264parse",
		"config-interval=1", // Częściej wysyłaj SPS/PPS
		"disable-passthrough=true",
		"!", "queue",
		"max-size-time=33333333", // ~1 frame at 30fps (w nanosekundach)
		"leaky=upstream",         // Zrzucaj stare ramki, nie nowe
		"!", "filesink", "location=/dev/stdout",
		"sync=false",
		"async=false",
		"buffer-mode=unbuffered", // Zmiana z 0 na unbuffered
	}

	// 	args = []string{
	// 	// "v4l2src", "device=/dev/video4",
	// 	// "!", "video/x-h264,width=1280,height=720,framerate=30/1",
	// 	// "!", "filesink", "location=/dev/stdout", "sync=false",
	// }

	// args = []string{
	// 	"v4l2src", "device=device=/dev/video4", "do-timestamp=true", "io-mode=2",
	// 	"!", "video/x-h264,width=1280,height=720,framerate=30/1",
	// 	"!", "h264parse", "config-interval=-1",
	// 	"!", "rtph264pay", "pt=96", "config-interval=-1",
	// 	"!", "rtph264depay",
	// 	"!", "filesink", "location=/dev/stdout", "sync=false", "buffer-mode=unbuffered",
	// }

	return exec.Command("gst-launch-1.0", args...)

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

// Funkcja do znajdowania NAL units w surowym strumieniu H.264
func findNALUnits(data []byte) [][]byte {
	var nalUnits [][]byte
	start := 0

	for i := 0; i < len(data)-3; i++ {
		// Szukamy start code: 0x00 0x00 0x00 0x01 lub 0x00 0x00 0x01
		if data[i] == 0x00 && data[i+1] == 0x00 {
			if (i+3 < len(data) && data[i+2] == 0x00 && data[i+3] == 0x01) ||
				(data[i+2] == 0x01) {

				// Jeśli to nie pierwszy NAL unit, zapisz poprzedni
				if start < i {
					nalUnits = append(nalUnits, data[start:i])
				}

				// Ustaw nowy start
				if data[i+2] == 0x00 && data[i+3] == 0x01 {
					start = i + 4 // 4-byte start code
					i += 3
				} else {
					start = i + 3 // 3-byte start code
					i += 2
				}
			}
		}
	}

	// Dodaj ostatni NAL unit
	if start < len(data) {
		nalUnits = append(nalUnits, data[start:])
	}

	return nalUnits
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

	ffmpegCmd := BuildFFmpegCommand(camera.Device, camera.Quality)
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

		log.Printf("📹 Rozpoczynam streaming H.264 dla kamery %s", camera.ID)

		var sps []byte
		var pps []byte
		var frameBuffer []byte
		var currentIsIDR bool
		buffer := make([]byte, 0, 64*1024) // Buffer dla danych

		flushFrame := func() {
			if len(frameBuffer) == 0 {
				return
			}
			// Dla IDR frame, dodaj SPS i PPS na początku
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

		readBuffer := make([]byte, 4096)
		for {
			n, err := stdout.Read(readBuffer)
			if err != nil {
				if err == io.EOF {
					log.Printf("📹 Koniec streamu dla kamery %s", camera.ID)
				} else {
					log.Printf("❌ Błąd odczytu strumienia: %v", err)
				}
				break
			}

			// Dodaj przeczytane dane do buffera
			buffer = append(buffer, readBuffer[:n]...)

			// Znajdź NAL units w buferze
			nalUnits := findNALUnits(buffer)

			// Zostaw ostatni fragment w buferze (może być niepełny)
			if len(nalUnits) > 0 {
				// Znajdź pozycję ostatniego NAL unit w buferze
				lastNALStart := bytes.LastIndex(buffer, []byte{0x00, 0x00, 0x00, 0x01})
				if lastNALStart == -1 {
					lastNALStart = bytes.LastIndex(buffer, []byte{0x00, 0x00, 0x01})
				}

				// Przetwórz wszystkie NAL units oprócz ostatniego (niepełnego)
				for _, nalData := range nalUnits[:len(nalUnits)-1] {
					if len(nalData) == 0 {
						continue
					}

					nalType := nalData[0] & 0x1F
					switch nalType {
					case 7: // SPS
						sps = nalData
						// log.Printf("📹 Otrzymano SPS dla kamery %s", camera.ID)
					case 8: // PPS
						pps = nalData
						// log.Printf("📹 Otrzymano PPS dla kamery %s", camera.ID)
					case 5: // IDR
						currentIsIDR = true
						frameBuffer = append([]byte{}, addStartCode(nalData)...)
						// frameBuffer = append(frameBuffer, addStartCode(nalData)...)
					case 1: // non-IDR slice
						frameBuffer = append(frameBuffer, addStartCode(nalData)...)
					case 9: // AUD - Access Unit Delimiter (koniec ramki)
						flushFrame()
					default: // Inne typy NAL units (np. SEI)
						frameBuffer = append(frameBuffer, addStartCode(nalData)...)
					}
				}

				// Zostaw fragment po ostatnim NAL unit w buferze
				if lastNALStart >= 0 {
					buffer = buffer[lastNALStart:]
				} else {
					buffer = buffer[:0]
				}
			}
		}

		// Flush ostatnia ramka jeśli istnieje
		flushFrame()
	}()

	return nil
}
