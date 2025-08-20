package structs

import (
	"net/http"
)

type ServerStatusResponse struct {
	Status bool `json:"status"`
}

type CameraStatusResponse struct {
	Status    bool `json:"status"`
	CameraNum int  `json:"camera_num"`
}

type CameraListResponse struct {
	ID       string `json:"id"`
	Device   string `json:"device"`
	Port     int    `json:"port"`
	IsActive bool   `json:"isActive"`
}

func SetCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")
}
