import React, { useRef, useState, useEffect } from "react";
import styles from './FourCamera.module.css';
import CameraPicker from "../../components/CameraPicker/CameraPicker";
import StreamViewer from "../../components/StreamViewer/StreamViewer";
import config from "../../config.js"


const FourCamera = () => {
  const [camport1, setCamport1] = useState(8081);
  const [camname1, setCamname1] = useState("camera_0")
  const [camport2, setCamport2] = useState(8081);
  const [camname2, setCamname2] = useState("camera_1")
  const [camport3, setCamport3] = useState(8082);
  const [camname3, setCamname3] = useState("camera_2")
  const [camport4, setCamport4] = useState(8083);
  const [camname4, setCamname4] = useState("camera_3")
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
      <h2>MULTIPLE CAMERA VIEW</h2>
      <ul className={styles.camerapicker_list}>
        <CameraPicker title="1" cameras={cameras} setCamport={setCamport1} setCamname={setCamname1} />
        <CameraPicker title="2" cameras={cameras} setCamport={setCamport2} setCamname={setCamname2} />
        <CameraPicker title="3" cameras={cameras} setCamport={setCamport3} setCamname={setCamname3} />
        <CameraPicker title="4" cameras={cameras} setCamport={setCamport4} setCamname={setCamname4} />
      </ul>

      <div className={styles.streamgrid}>
        <StreamViewer monitor="monitor 1" camname={camname1} camport={camport1} x_size={700} y_size={500} />
        <StreamViewer monitor="monitor 2" camname={camname2} camport={camport2} x_size={700} y_size={500} />
        <StreamViewer monitor="monitor 3" camname={camname3} camport={camport3} x_size={700} y_size={500} />
        <StreamViewer monitor="monitor 4" camname={camname4} camport={camport4} x_size={700} y_size={500} />
      </div>
    </div>
  );
};

export default FourCamera;
