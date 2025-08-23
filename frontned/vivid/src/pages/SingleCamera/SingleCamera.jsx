import React, { useRef, useState, useEffect } from "react";
import styles from './SingleCamera.module.css';
import CameraPicker from "../../components/CameraPicker/CameraPicker";
import StreamViewer from "../../components/StreamViewer/StreamViewer";
import config from "../../config.js"

const SingleCamera = () => {
  const [camport, setCamport] = useState(8081);
  const [camname, setCamname] = useState("camera_0")
  const [cameras, setCameras] = useState([]);

  useEffect(() => {
    const fetchCameras = async () => {
      try {
        const response = await fetch(`http://${config.IPADDR}:${config.PORT}/api/cameras`);
        const data = await response.json();
        setCameras(data);
      } catch (err) {
        console.error("Błąd pobierania kamer:", err);
      }
    };

    fetchCameras();
  }, []);


  return (
    <div className={styles.wrapper}>
      <h2>SINGLE CAMERA VIEW</h2>
      <CameraPicker title="1" cameras={cameras} setCamport={setCamport} setCamname={setCamname} />
      <StreamViewer camname={camname} camport={camport} x_size={1600} y_size={1000} />
    </div>
  );
};

export default SingleCamera;
