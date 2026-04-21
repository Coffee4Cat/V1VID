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
	fmt.Printf("%d\n", mode)

	switch mode {
	case 4: // MJPEG
		args = []string{
			"v4l2src", "device=" + device,
			"do-timestamp=true",
			"!", "image/jpeg,width=1024,height=576,framerate=20/1",
			"!", "jpegdec",
			"!", "videoconvert",
			"!", "video/x-raw,format=I420",
			"!", "queue",
			"max-size-buffers=3",
			"max-size-time=0",
			"max-size-bytes=0",
			"leaky=upstream",
			"!", "x264enc",
			"tune=zerolatency",
			"speed-preset=ultrafast",
			"bitrate=4000",
			"key-int-max=30",
			"bframes=0",
			"aud=true",
			"sliced-threads=false",
			"sync-lookahead=0",
			"rc-lookahead=0",
			"vbv-buf-capacity=120",
			"threads=1",
			"!", "video/x-h264,profile=baseline,stream-format=byte-stream",
			"!", "h264parse",
			"config-interval=1",
			"!", "queue",
			"max-size-buffers=2",
			"leaky=upstream",
			"!", "filesink", "location=/dev/stdout",
			"sync=false",
			"async=false",
			"buffer-mode=unbuffered",
			"blocksize=4096",
		}
	default: // x264
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
	var starts []int

	i := 0
	for i < len(data)-2 {
		if data[i] == 0x00 && data[i+1] == 0x00 {
			if i+3 < len(data) && data[i+2] == 0x00 && data[i+3] == 0x01 {
				starts = append(starts, i+4)
				i += 4
				continue
			}
			if data[i+2] == 0x01 {
				starts = append(starts, i+3)
				i += 3
				continue
			}
		}
		i++
	}

	for idx, start := range starts {
		end := len(data)
		if idx+1 < len(starts) {
			end = starts[idx+1]
			if end >= 4 && data[end-4] == 0x00 {
				end -= 4
			} else if end >= 3 && data[end-3] == 0x00 {
				end -= 3
			}
		}
		if start < end {
			nalUnits = append(nalUnits, data[start:end])
		}
	}

	return nalUnits
}

const (
	maxFrameSize          = 512 * 1024 // 512 KB
	maxFrameLatency       = 150 * time.Millisecond
	fallbackFrameDuration = time.Second / 30
)

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

	ffmpegCmd := BuildFFmpegCommand(camera.CurrentDevice, camera.Quality)

	stdout, err := ffmpegCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("[ERROR] stdoutpipe: %v", err)
	}

	if err := ffmpegCmd.Start(); err != nil {
		return fmt.Errorf("[ERROR] GSTREAMER failure: %v", err)
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

		log.Printf("[STREAM] Camera %s", camera.ID)

		var sps []byte
		var pps []byte
		var frameBuffer []byte
		var currentIsIDR bool

		lastFrameTime := time.Now()
		buffer := make([]byte, 0, 256*1024)
		readBuffer := make([]byte, 65536)

		flushFrame := func() {
			if len(frameBuffer) == 0 {
				return
			}

			now := time.Now()
			elapsed := now.Sub(lastFrameTime)

			if !currentIsIDR && elapsed > maxFrameLatency {
				log.Printf("[DROP] Camera %s: dropping non-IDR frame, latency=%v", camera.ID, elapsed)
				frameBuffer = nil
				currentIsIDR = false
				return
			}

			duration := elapsed
			if duration < 10*time.Millisecond || duration > 200*time.Millisecond {
				duration = fallbackFrameDuration
			}
			lastFrameTime = now

			if currentIsIDR && sps != nil && pps != nil {
				out := make([]byte, 0, len(sps)+len(pps)+len(frameBuffer)+12)
				out = append(out, addStartCode(sps)...)
				out = append(out, addStartCode(pps)...)
				out = append(out, frameBuffer...)
				frameBuffer = out
			}

			sample := media.Sample{
				Data:     frameBuffer,
				Duration: duration,
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

		for {
			n, err := stdout.Read(readBuffer)
			if err != nil {
				if err == io.EOF {
					log.Printf("[STREAM] End of Stream for camera %s", camera.ID)
				}
				break
			}

			buffer = append(buffer, readBuffer[:n]...)

			if len(buffer) > 4*maxFrameSize {
				log.Printf("[WARN] Camera %s: work buffer overflow (%d bytes), resetting", camera.ID, len(buffer))
				buffer = buffer[:0]
				frameBuffer = nil
				currentIsIDR = false
				continue
			}

			nalUnits := findNALUnits(buffer)

			if len(nalUnits) > 0 {
				processUpTo := len(nalUnits) - 1

				for _, nalData := range nalUnits[:processUpTo] {
					if len(nalData) == 0 {
						continue
					}

					if len(frameBuffer)+len(nalData)+4 > maxFrameSize {
						log.Printf("[WARN] Camera %s: frameBuffer overflow, dropping frame", camera.ID)
						frameBuffer = nil
						currentIsIDR = false
						continue
					}

					nalType := nalData[0] & 0x1F
					switch nalType {
					case 7: // SPS
						sps = make([]byte, len(nalData))
						copy(sps, nalData)
					case 8: // PPS
						pps = make([]byte, len(nalData))
						copy(pps, nalData)
					case 5: // IDR
						flushFrame()
						currentIsIDR = true
						frameBuffer = append([]byte{}, addStartCode(nalData)...)
					case 1: // non-IDR slice
						frameBuffer = append(frameBuffer, addStartCode(nalData)...)
					case 9: // AUD - Access Unit Delimiter
						flushFrame()
					default:
						frameBuffer = append(frameBuffer, addStartCode(nalData)...)
					}
				}

				lastSC4 := bytes.LastIndex(buffer, []byte{0x00, 0x00, 0x00, 0x01})
				lastSC3 := bytes.LastIndex(buffer, []byte{0x00, 0x00, 0x01})

				lastNALStart := lastSC4
				if lastSC3 > lastSC4 {
					if lastSC3 == 0 || buffer[lastSC3-1] != 0x00 {
						lastNALStart = lastSC3
					}
				}

				if lastNALStart >= 0 {
					buffer = buffer[lastNALStart:]
				} else {
					buffer = buffer[:0]
				}
			}
		}

		flushFrame()
	}()

	return nil
}
