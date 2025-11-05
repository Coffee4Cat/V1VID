package structs

import (
	"net/http"
	"os/exec"
	"sync"

	"github.com/pion/webrtc/v3"
)

// Dodać structure związany z Devicem
// będzie on miał EveryDevice, MJPGDevice, H264Device
// A Camera będzie miała CurrentDevice (string) oraz currentMode(const enum)

type Camera struct {
	ID       string
	Device   string
	Port     int
	Track    *webrtc.TrackLocalStaticSample
	FFmpeg   *exec.Cmd
	IsActive bool
	Server   *http.Server
	MMutex   sync.RWMutex
	Quality  int
	PipePath string
}

type CameraManager struct {
	Cameras map[string]*Camera
	MMutex  sync.RWMutex
}

var Manager = &CameraManager{
	Cameras: make(map[string]*Camera),
}
