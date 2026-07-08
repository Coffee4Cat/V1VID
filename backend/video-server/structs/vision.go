package structs

import (
	"fmt"
	"net/http"
	"os/exec"
	"sync"

	"github.com/pion/webrtc/v3"
)

// Dodać structure związany z Devicem
// będzie on miał EveryDevice, MJPGDevice, H264Device
// A Camera będzie miała CurrentDevice (string) oraz currentMode(const enum)

type CameraType int

const (
	MJPEG = iota
	H264
)

type Devices struct {
	EveryDevice []string
	MJPGDevice  string
	H264Device  string
}

type Camera struct {
	ID            string
	Name          string
	CamType       CameraType
	Devs          Devices
	CurrentDevice string
	Port          int
	Track         *webrtc.TrackLocalStaticSample
	FFmpeg        *exec.Cmd
	IsActive      bool
	Server        *http.Server
	MMutex        sync.RWMutex
	Quality       int
	PipePath      string
}

func (c *Camera) Represent() {
	fmt.Printf("%s ", c.Name)
	switch c.CamType {
	case MJPEG:
		fmt.Printf("[%s MJPG]\n", c.Devs.MJPGDevice)
	case H264:
		fmt.Printf("[%s MJPG, ", c.Devs.MJPGDevice)
		fmt.Printf("%s H264]\n", c.Devs.H264Device)
	}
	fmt.Println()
}
func MapDevices(cam *Camera, camtype CameraType) {
	switch camtype {
	case MJPEG:
		cam.Devs.MJPGDevice = cam.Devs.EveryDevice[0]
	case H264:
		cam.Devs.MJPGDevice = cam.Devs.EveryDevice[0]
		cam.Devs.H264Device = cam.Devs.EveryDevice[2]
	}
}
func ClasifyCamera(camera *Camera) {
	if len(camera.Devs.EveryDevice) >= 4 {
		camera.CamType = H264
	} else if len(camera.Devs.EveryDevice) >= 1 {
		camera.CamType = MJPEG
	} else {
		return
	}
	MapDevices(camera, camera.CamType)
}

type CameraManager struct {
	Cameras map[string]*Camera
	MMutex  sync.RWMutex
}

var Manager = &CameraManager{
	Cameras: make(map[string]*Camera),
}
