import React from "react";
import styles from './Navigator.module.css';
import logo from '../../assets/logo.png';

function Navigator() {
  return (
    <nav className={styles.navbar}>
        <div className={styles.logo_container}>
            <img src={logo} alt="Logo" className={styles.logo_img} />
            <div className={styles.logo_text}>V1VID</div>
        </div>
        <ul className={styles.menu}>
            <li><a href="#home">Home</a></li>
            <li><a href="#videomodes">Video Modes</a></li>
            <li><a href="#camerastatus">Camera Status</a></li>
            <li><a href="#about">About</a></li>
        </ul>
    </nav>
  );
}

export default Navigator;