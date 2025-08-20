# V1VID
### Use Case
**V1VID** is a web application designed for low-latency camera feed streaming over WEBRTC protocol.


### Tech stack
![Frontend](https://img.shields.io/badge/frontend-react-cyan?style=for-the-badge)  
![Backend](https://img.shields.io/badge/backend-go-cyan?style=for-the-badge)  
![OperatingSystem](https://img.shields.io/badge/os-ubuntu_24.04-orange?style=for-the-badge)  
![Protocol](https://img.shields.io/badge/protocol-WEBRTC-lime?style=for-the-badge)  


### How To Set Up?
> [!NOTE]
> Guide only for setting up tech stack, not the 'systemd' server
#### Frontend
```bash
sudo apt update  
sudo apt install nodejs npm -y
cd (frontend-catalogue)/
npm install
npm run build
serve -s build -l 5000
```
#### Backend
```bash
sudo apt update
sudo apt install golang-go -y
export PATH=$PATH:/usr/local/go/bin
source ~/.bashrc
cd (backend-catalogue)/video-server
go run main.go
```






