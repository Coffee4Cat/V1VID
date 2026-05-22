import React, { useRef, useState } from "react";
import styles from './StreamViewer.module.css';
import config from "../../config.js"


const StreamViewer = ({monitor, camname, camport, x_size, y_size}) => {
    const videoRef = useRef(null);
    const pcRef = useRef(null);
    const wsRef = useRef(null);
    const [started, setStarted] = useState(false);
    const [rotation, setRotation] = useState(0);
    const [isHovered, setIsHovered] = useState(false);

    const rotate = () => setRotation(prev => (prev + 90) % 360);

    const startCamera = () => {
    if (started) return;
    setStarted(true);
    const camera_ws = `ws://${config.IPADDR}:` + camport + "/ws";
    const ws = new WebSocket(camera_ws);
    wsRef.current = ws;

    const pc = new RTCPeerConnection({ iceServers: [] });
    pcRef.current = pc;

    pc.ontrack = (event) => {
        if (videoRef.current) {
        videoRef.current.srcObject = event.streams[0];
        }
    };

    pc.onicecandidate = (event) => {
        if (event.candidate) {
        ws.send(JSON.stringify({ type: "ice-candidate", data: event.candidate }));
        }
    };

    ws.onopen = () => {
        console.log("🔌 Połączono z WebSocket, wysyłam viewer");
        ws.send(JSON.stringify({ type: "viewer" }));
    };

    ws.onmessage = async (message) => {
        const msg = JSON.parse(message.data);
        switch (msg.type) {
        case "offer":
            await pc.setRemoteDescription(msg.data);
            const answer = await pc.createAnswer();
            await pc.setLocalDescription(answer);
            ws.send(JSON.stringify({ type: "answer", data: answer }));
            break;

        case "ice-candidate":
            try {
            await pc.addIceCandidate(msg.data);
            } catch (err) {
            console.error("❌ Błąd dodawania ICE candidate:", err);
            }
            break;

        default:
            console.log("Nieznany typ wiadomości:", msg.type);
        }
    };

    ws.onclose = () => console.log("🔌 WebSocket zamknięty");
    };

    const stopCamera = () => {
    if (pcRef.current) {
        pcRef.current.close();
        pcRef.current = null;
    }
    if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
    }
    if (videoRef.current) {
        videoRef.current.srcObject = null;
    }
    setStarted(false);
    };

    return (
        <div className={styles.wrapper}>
            {!started && (
                <button onClick={startCamera}>Start Watching <strong>{camname}</strong></button>
            )}
            <div
                className={styles.container}
                onMouseEnter={() => setIsHovered(true)}
                onMouseLeave={() => setIsHovered(false)}
            >
                <video
                    ref={videoRef}
                    autoPlay
                    playsInline
                    muted
                    width={x_size}
                    height={y_size}
                    style={{ transform: `rotate(${rotation}deg)`, transition: 'transform 0.3s ease' }}
                />
                {isHovered && (
                    <button className={styles.rotateBtn} onClick={rotate} title="Obróć o 90°">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round">
                            <path d="M21 2v6h-6"/>
                            <path d="M21 8A9 9 0 1 0 18.6 18.6"/>
                        </svg>
                        <span>90°</span>
                    </button>
                )}
            </div>
        </div>
    );
}


export default StreamViewer;