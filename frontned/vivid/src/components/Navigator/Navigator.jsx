import React from "react";
import { Link }  from "react-router-dom";
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
            <li><Link to="/">Home</Link></li>
            <li><Link to="/videomodes">Video Modes</Link></li>
            <li><Link to="/camerastatus">Camera Status</Link></li>
            <li><Link to="/about">About</Link></li>
        </ul>
    </nav>
  );
}

export default Navigator;