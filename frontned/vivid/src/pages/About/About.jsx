import React from "react";
import styles from "./About.module.css"



function About() {
  return (
    <div className={styles.wrapper}>
      <div className={`${styles['bg-blob']} ${styles['bg-blob1']}`}></div>
      <div className={`${styles['bg-blob']} ${styles['bg-blob2']}`}></div>
      <div className={`${styles['bg-blob']} ${styles['bg-blob3']}`}></div>
      <div className={`${styles['bg-blob']} ${styles['bg-blob4']}`}></div>
      <div className={styles.title}>
        <h1>About V1VID</h1> 
      </div>
      <div className={styles.text}>
        <p>
            V1VID is a cutting-edge solution designed for low-latency video streaming, specifically tailored for HAL-062.
            Our platform ensures seamless video delivery with minimal delay, making it ideal for real-time applications.
        </p>
        <p>
            Camera feed, encoded in h.264 is streamed via SRT protocol, enabling fast and reliable work.    
        </p>
        <p>
            For more information, please contact the Autonomy departpemt.
        </p>
      </div>
    </div>
  );
}

export default About;