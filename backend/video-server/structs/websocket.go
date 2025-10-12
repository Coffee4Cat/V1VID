package structs

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var BasePort = 8081
var Upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type SignalingMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type SafeWebSocketConn struct {
	Conn     *websocket.Conn
	SendChan chan SignalingMessage
	Done     chan struct{}
	Once     sync.Once
}

func NewSafeWebSocketConn(conn *websocket.Conn) *SafeWebSocketConn {
	safeConn := &SafeWebSocketConn{
		Conn:     conn,
		SendChan: make(chan SignalingMessage, 256),
		Done:     make(chan struct{}),
	}

	go safeConn.WritePump()

	return safeConn
}

func (s *SafeWebSocketConn) WritePump() {
	defer s.Conn.Close()

	ticker := time.NewTicker(54 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case message := <-s.SendChan:
			s.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := s.Conn.WriteJSON(message); err != nil {
				log.Printf("[ERROR] Websocket failure: %v", err)
				return
			}

		case <-ticker.C:
			s.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := s.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		case <-s.Done:
			return
		}
	}
}

func (s *SafeWebSocketConn) SendMessage(msg SignalingMessage) error {
	select {
	case s.SendChan <- msg:
		return nil
	case <-s.Done:
		return nil
	default:
		return nil
	}
}

func (s *SafeWebSocketConn) Close() {
	s.Once.Do(func() {
		close(s.Done)
		s.Conn.Close()
	})
}
