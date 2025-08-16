// import React from "react";
// import styles from "./DualCamera.module.css"



// function DualCamera() {

//     return (
//         <div className={styles.wrapper}>
//             <div className={styles.title}>
//                 <br/>
//                 <p>Pick the double-trouble of your choice</p>
//             </div>
//         </div>
//     );

// };

// export default DualCamera

import React, { useEffect, useRef } from "react";

export default function FourCamera() {
  const videoRef = useRef(null);
  const socketRef = useRef(null);
  const peerConnectionRef = useRef(null);

  useEffect(() => {
    // Połączenie WebSocket zamiast Socket.IO
    socketRef.current = new WebSocket('ws://localhost:8080/ws');
    console.log("start");

    const peerConnection = new RTCPeerConnection({
      iceServers: [
        { urls: "stun:stun.l.google.com:19302" } // Dodaj STUN serwer
      ],
    });
    peerConnectionRef.current = peerConnection;

    peerConnection.ontrack = (event) => {
      console.log("Otrzymano track", event);
      if (videoRef.current) {
        videoRef.current.srcObject = event.streams[0];
      }
    };

    peerConnection.onicecandidate = (event) => {
      if (event.candidate) {
        // Wysyłanie jako JSON z typem wiadomości
        socketRef.current.send(JSON.stringify({
          type: "ice-candidate",
          data: event.candidate
        }));
      }
    };

    // Obsługa wiadomości WebSocket
    socketRef.current.onmessage = async (event) => {
      const message = JSON.parse(event.data);
      
      switch (message.type) {
        case "offer":
          console.log("Otrzymano offer");
          await peerConnection.setRemoteDescription(message.data);
          const answer = await peerConnection.createAnswer();
          await peerConnection.setLocalDescription(answer);
          
          socketRef.current.send(JSON.stringify({
            type: "answer",
            data: answer
          }));
          break;
          
        case "ice-candidate":
          console.log("Otrzymano ICE candidate");
          try {
            await peerConnection.addIceCandidate(message.data);
          } catch (e) {
            console.error("Błąd dodawania ICE candidate:", e);
          }
          break;
          
        default:
          console.log("Nieznany typ wiadomości:", message.type);
      }
    };

    socketRef.current.onopen = () => {
      console.log("WebSocket połączony");
      // Informujemy serwer, że jesteśmy viewer
      socketRef.current.send(JSON.stringify({
        type: "viewer",
        data: {}
      }));
    };

    socketRef.current.onerror = (error) => {
      console.error("WebSocket error:", error);
    };

    return () => {
      if (peerConnectionRef.current) {
        peerConnectionRef.current.close();
      }
      if (socketRef.current) {
        socketRef.current.close();
      }
    };
  }, []);

  return (
    <div>
      <h2>Podgląd kamery</h2>
      <video
        ref={videoRef}
        autoPlay
        playsInline
        muted
        controls
        style={{ width: "100%", maxWidth: 640 }}
      />
    </div>
  );
}