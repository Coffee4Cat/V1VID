import React from "react";
import styles from "./About.module.css"
import logo from '../../assets/logo.png';
import knr from "../../assets/knr.png"


function About() {
  return (
    <div className={styles.wrapper}>
      <div className={`${styles['bg-blob']} ${styles['bg-blob1']}`}></div>
      <div className={`${styles['bg-blob']} ${styles['bg-blob2']}`}></div>
      <div className={styles.title}>
        <h1>About V1VID</h1> 
      </div>
      <div className={styles.text}>
        <h3>What is <strong>V1VID</strong>?</h3>
        <p>
            + <strong>V1VID</strong> is a cutting-edge solution designed for low-latency video streaming, specifically tailored for HAL-062.
            The platform ensures seamless video delivery with minimal delay, making it ideal for real-time applications.
        </p>
        <h3>How to use it?</h3>
        <p>
            <strong>+</strong> <strong>Home</strong> - there you can prompt the video server to see if it works. If it doesn't, well... Eather the connection is lost or I just fkcd sth up. 
            These indicators are there just to test the server about it's "alive" state (it can eather be alive or not).<br/>
            <strong>+</strong> <strong>Camera Status</strong> - There you <strong>will</strong> be able to check the general status of the pluged cameras. 
            The main point is to even see what are the exact cameras that work within the system at certain point in time.<br/>
            <strong>+</strong> <strong>Video Modes</strong> - There you pick you video mode. Basic options are: <i>single-camera</i>, <i>double-camera</i>, <i>every-camera</i>. 
            From there on the general rule is - pick the camera, receive the livesteam, and watch the rover go!
        </p>
        <h3>How it works?</h3>
        <p>
            + Backend server plays as a camera-system driver. Server, based on received command, starts stream/stops stream/restarts stream/changes settings of the cameras.
            User (via webapp) can decide control each and every camera pluged into the system.     
        </p>
        <h3>Is there anythin else?</h3>
        <p>
            + For more detailed information, please contact the <strong>Autonomy</strong> departpemt.
        </p>
      </div>
      <div className={styles.logo_container}>
          <img src={logo} alt="Logo" className={styles.logo_img} />
          <img src={knr} alt="Logo" className={styles.logo_img_knr} />
      </div>
    </div>
  );
}

export default About;