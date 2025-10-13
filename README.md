# V1VID
### Short description
**V1VID** is a modern web application designed for low-latency livestreaming from multiple USB cameras with minimal CPU usage, providing smooth and efficient real-time video transmission directly to web browsers using WebRTC.

### Tech stack
![Frontend](https://img.shields.io/badge/frontend-react-cyan?style=for-the-badge)  
![Backend](https://img.shields.io/badge/backend-golang-cyan?style=for-the-badge)   
![Protocol](https://img.shields.io/badge/protocol-WEBRTC-orange?style=for-the-badge)  

### How it works
> [!IMPORTANT]
> System is designed for LAN

The system is designed to stream remote camera feed with minimal latency and CPU usage using USB cameras supporting h264 hardware encoding. The frontend connects to the backend over WebRTC and establishes a real-time video session.


### How to use it
1. Plug your usb cameras to the server
2. Put their */dev/video* ID into **camera.go** (TODO: Automation)
3. Start the backend server and serve the frontend.
4. If works, the control diode on the home page should be colorfull (gray = not connected) 

### Task List
- **General**
    - [x] functional video server (backend)
    - [x] functional stream receiver (frontend)
    - [ ] automated device detection and naming
    - [x] support for h264 devices
    - [ ] support for mjpeg devices

- **Stream**
    - [ ] manual video resolution adjustment 
    - [x] manual video parameter adjustment for lower-latency of hardware encoded h264 stream (Indor, Sunny, Cloudy)
- **Telemetry**
    - [ ] CPU usage telemetry (for mjpeg support)



## License

This project is licensed under the [MIT License](./LICENSE).

### Third-Party Licenses

This software includes open-source components:
- [pion/webrtc](https://github.com/pion/webrtc) — MIT License  
- [gorilla/websocket](https://github.com/gorilla/websocket) — BSD 2-Clause License

All third-party libraries retain their original licenses and copyright notices.





