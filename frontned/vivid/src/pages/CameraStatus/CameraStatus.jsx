import React, { useEffect, useState } from "react";
import styles from './CameraStatus.module.css';
import CameraActivator from "../../components/CameraActivator/CameraActivator";
import config from "../../config.js"

function CameraStatus() {
  const [cameras, setCameras] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchCameras = async () => {
      try {
        const response = await fetch(`http://${config.IPADDR}:${config.PORT}/api/cameras`);
        const data = await response.json();
        setCameras(data);
      } catch (err) {
        console.error("Błąd pobierania kamer:", err);
      } finally {
        setLoading(false);
      }
    };

    fetchCameras();
  }, []);

  return (
    <div className={styles.wrapper}>
      <div className={`${styles['bg-blob']} ${styles['bg-blob1']}`}></div>
      <div className={`${styles['bg-blob']} ${styles['bg-blob2']}`}></div>

      <div className={styles.title}>
        <h1>Camera Status</h1>
      </div>

      <div className={styles.description}>
        <p>Check telemetry data related to cameras</p>
        <p>Camera count: {cameras.length}</p>
      </div>

      <div className={styles.panel}>
        {loading ? (
          <p>Loading cameras...</p>
        ) : (
          <ul className={styles.blocklist}>
            {cameras.map((camera) => (
                <CameraActivator text={camera.id} address={`http://${config.IPADDR}:${config.PORT}/api/camera`} camera_id={camera.id} entry_status={camera.isActive} entry_quality={camera.quality}/>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}

export default CameraStatus;