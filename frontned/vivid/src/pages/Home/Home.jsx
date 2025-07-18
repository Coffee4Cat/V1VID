import React from "react";
import styles from './Home.module.css';

function Home() {
  return (
    <div className={styles.wrapper}>
        <div className={`${styles['bg-blob']} ${styles['bg-blob1']}`}></div>
        <div className={`${styles['bg-blob']} ${styles['bg-blob2']}`}></div>
        <div className={`${styles['bg-blob']} ${styles['bg-blob3']}`}></div>
        <div className={`${styles['bg-blob']} ${styles['bg-blob4']}`}></div>
        <div className={styles.title}>
        <h1>V1VID</h1>
        </div>
        <div className={styles.description}>
          <p>Top One Solution for Low-Latency Video Streaming for HAL-062</p>
        </div>
    </div>
  );
}

export default Home;
