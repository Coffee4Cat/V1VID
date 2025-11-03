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

func BuildV4L2Command(device string, mode int) {
	switch mode {
	case 1: // INDOR
		exec.Command("v4l2-ctl", "-d", device, "-c", "auto_exposure=1").Run()
		time.Sleep(2200 * time.Millisecond)
		exec.Command("v4l2-ctl", "-d", device, "-c", "exposure_time_absolute=250").Run()
		time.Sleep(2200 * time.Millisecond)
		exec.Command("v4l2-ctl", "-d", device, "-c", "white_balance_automatic=0").Run()
		time.Sleep(2200 * time.Millisecond)
		exec.Command("v4l2-ctl", "-d", device, "-c", "white_balance_absolute=7500").Run()
	case 2: // CLOUDY
		exec.Command("v4l2-ctl", "-d", device, "-c", "auto_exposure=1").Run()
		time.Sleep(2200 * time.Millisecond)
		exec.Command("v4l2-ctl", "-d", device, "-c", "exposure_time_absolute=50").Run()
		time.Sleep(2200 * time.Millisecond)
		exec.Command("v4l2-ctl", "-d", device, "-c", "white_balance_automatic=0").Run()
		time.Sleep(2200 * time.Millisecond)
		exec.Command("v4l2-ctl", "-d", device, "-c", "white_balance_absolute=8000").Run()
	case 3: // SUNNY
		exec.Command("v4l2-ctl", "-d", device, "-c", "auto_exposure=1").Run()
		time.Sleep(2200 * time.Millisecond)
		exec.Command("v4l2-ctl", "-d", device, "-c", "exposure_time_absolute=20").Run()
		time.Sleep(2200 * time.Millisecond)
		exec.Command("v4l2-ctl", "-d", device, "-c", "white_balance_automatic=0").Run()
		time.Sleep(2200 * time.Millisecond)
		exec.Command("v4l2-ctl", "-d", device, "-c", "white_balance_absolute=8000").Run()
	}

}

func BuildFFmpegCommand(device string, mode int) *exec.Cmd {
	var args []string
	// 800x600 OR 1280x720
	switch mode {
	case 1: // INDOR
		args = []string{
			"v4l2src", "device=" + device,
			"do-timestamp=true",
			"io-mode=2",
			"!", "video/x-h264,width=1280,height=720,framerate=30/1",
			"!", "h264parse",
			"config-interval=1",
			"disable-passthrough=true",
			"!", "queue",
			"max-size-time=33333333",
			"leaky=upstream",
			"!", "filesink", "location=/dev/stdout",
			"sync=false",
			"async=false",
			"buffer-mode=unbuffered",
		}
	case 2: // SUNNY
		args = []string{
			"v4l2src", "device=" + device,
			"do-timestamp=true",
			"io-mode=2",
			"!", "video/x-h264,width=1280,height=720,framerate=30/1",
			"!", "h264parse",
			"config-interval=1",
			"disable-passthrough=true",
			"!", "queue",
			"max-size-time=33333333",
			"leaky=upstream",
			"!", "filesink", "location=/dev/stdout",
			"sync=false",
			"async=false",
			"buffer-mode=unbuffered",
		}

	case 3:
		args = []string{
			"v4l2src", "device=" + device,
			"!", "image/jpeg,framerate=30/1,width=1280,height=720",
			"!", "jpegdec",
			"!", "videoconvert",
			"!", "x264enc",
			"tune=zerolatency",
			"bitrate=4000",
			"speed-preset=fast",
			"key-int-max=30",
			"!", "video/x-h264,profile=baseline",
			"!", "h264parse",
			"!", "filesink", "location=/dev/stdout",
		}

	}

	return exec.Command("gst-launch-1.0", args...)

}

func HandleCameraWebSocket(w http.ResponseWriter, r *http.Request, camera *structs.Camera, config webrtc.Configuration) {
	conn, err := structs.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[ERROR] Websocket error for camera %s: %v", camera.ID, err)
		return
	}

	safeConn := structs.NewSafeWebSocketConn(conn)
	defer safeConn.Close()

	log.Printf("New connection for camera %s", camera.ID)

	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		log.Printf("PeerConnection error for camera %s: %v", camera.ID, err)
		return
	}
	defer peerConnection.Close()

	camera.MMutex.RLock()
	if camera.IsActive && camera.Track != nil {
		if _, err := peerConnection.AddTrack(camera.Track); err != nil {
			log.Printf("[ERROR] PeerConnection failure for camera %s: %v", camera.ID, err)
		} else {
			log.Printf("Added Camera %s to PeerConnection", camera.ID)
		}
	}
	camera.MMutex.RUnlock()

	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}
		candidateMsg := structs.SignalingMessage{
			Type: "ice-candidate",
			Data: candidate.ToJSON(),
		}
		safeConn.SendMessage(candidateMsg)
	})

	peerConnection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("State of Camera %s: %s", camera.ID, state.String())
	})

	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var msg structs.SignalingMessage
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		if err := conn.ReadJSON(&msg); err != nil {
			log.Printf("[ERROR] Websocket error for camera %s: %v", camera.ID, err)
			break
		}

		switch msg.Type {
		case "viewer":
			offer, err := peerConnection.CreateOffer(nil)
			if err != nil {
				log.Printf("[ERROR] Offer failure for camera %s: %v", camera.ID, err)
				continue
			}

			if err := peerConnection.SetLocalDescription(offer); err != nil {
				log.Printf("[ERROR] Local description failure for camera %s: %v", camera.ID, err)
				continue
			}

			offerMsg := structs.SignalingMessage{Type: "offer", Data: offer}
			if err := safeConn.SendMessage(offerMsg); err != nil {
				log.Printf("[ERROR] Offer failure for camera %s: %v", camera.ID, err)
			} else {
				log.Printf("Camera %s OFFER", camera.ID)
			}

		case "answer":
			answerData, _ := json.Marshal(msg.Data)
			var answer webrtc.SessionDescription
			json.Unmarshal(answerData, &answer)
			peerConnection.SetRemoteDescription(answer)
			log.Printf("Camera %s ANSWER", camera.ID)

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

func findNALUnits(data []byte) [][]byte {
	var nalUnits [][]byte
	start := 0

	for i := 0; i < len(data)-3; i++ {
		//start code: 0x00 0x00 0x00 0x01 lub 0x00 0x00 0x01
		if data[i] == 0x00 && data[i+1] == 0x00 {
			if (i+3 < len(data) && data[i+2] == 0x00 && data[i+3] == 0x01) ||
				(data[i+2] == 0x01) {

				if start < i {
					nalUnits = append(nalUnits, data[start:i])
				}

				if data[i+2] == 0x00 && data[i+3] == 0x01 {
					start = i + 4
					i += 3
				} else {
					start = i + 3
					i += 2
				}
			}
		}
	}

	if start < len(data) {
		nalUnits = append(nalUnits, data[start:])
	}

	return nalUnits
}

func StartCameraStream(camera *structs.Camera) error {
	camera.MMutex.Lock()
	defer camera.MMutex.Unlock()

	if camera.IsActive {
		return fmt.Errorf("Camera %s is active", camera.ID)
	}

	h264Track, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{MimeType: "video/H264"},
		"video",
		camera.ID,
	)
	if err != nil {
		return fmt.Errorf("[ERROR] %v", err)
	}
	camera.Track = h264Track

	ffmpegCmd := BuildFFmpegCommand(camera.Device, camera.Quality)

	stdout, err := ffmpegCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("[ERROR] stdoutpipe: %v", err)
	}

	if err := ffmpegCmd.Start(); err != nil {
		return fmt.Errorf("[ERROR] GSTREAMER failure: %v", err)
	}
	time.Sleep(10000 * time.Millisecond)
	// BuildV4L2Command(camera.Device, camera.Quality)

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

		log.Printf("[STREAM] Camera %s", camera.ID)

		var sps []byte
		var pps []byte
		var frameBuffer []byte
		var currentIsIDR bool
		buffer := make([]byte, 0, 64*1024)

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
					log.Printf("[DEVICE] Track closed for camera %s", camera.ID)
					return
				}
			}

			frameBuffer = nil
			currentIsIDR = false
		}

		readBuffer := make([]byte, 4096)
		for {
			n, err := stdout.Read(readBuffer)
			if err != nil {
				if err == io.EOF {
					log.Printf("[STREAM] End of Stream for camera %s", camera.ID)
				}
				break
			}

			buffer = append(buffer, readBuffer[:n]...)

			nalUnits := findNALUnits(buffer)

			if len(nalUnits) > 0 {
				lastNALStart := bytes.LastIndex(buffer, []byte{0x00, 0x00, 0x00, 0x01})
				if lastNALStart == -1 {
					lastNALStart = bytes.LastIndex(buffer, []byte{0x00, 0x00, 0x01})
				}

				for _, nalData := range nalUnits[:len(nalUnits)-1] {
					if len(nalData) == 0 {
						continue
					}

					nalType := nalData[0] & 0x1F
					switch nalType {
					case 7: // SPS
						sps = nalData
					case 8: // PPS
						pps = nalData
					case 5: // IDR
						currentIsIDR = true
						frameBuffer = append([]byte{}, addStartCode(nalData)...)
						// frameBuffer = append(frameBuffer, addStartCode(nalData)...)
					case 1: // non-IDR slice
						frameBuffer = append(frameBuffer, addStartCode(nalData)...)
					case 9: // AUD - Access Unit Delimiter
						flushFrame()
					default:
						frameBuffer = append(frameBuffer, addStartCode(nalData)...)
					}
				}

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
