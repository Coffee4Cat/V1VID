package main

import (
	"fmt"
	"time"

	"github.com/pion/webrtc/v3"
)

func main() {
	s := webrtc.Sample{
		Data:     []byte{0xDE, 0xAD, 0xBE, 0xEF},
		Duration: time.Second,
	}
	fmt.Printf("Sample: %+v\n", s)
}
