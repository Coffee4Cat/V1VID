import React, { useRef, useState, useEffect } from "react";
import styles from './CameraPicker.module.css';




const CameraPicker = ({title, cameras, setCamport, setCamname}) => {



    return (
        <div className={styles.container}>
            <div className={styles.title}>
                <h3>Pick Stream for monitor {title}</h3>
            </div>
            <ul className={styles.camera_list}>
                {cameras.map((camera) => (
                <li key={camera.port}>
                    <button className={ camera.isActive ? styles.button_active : styles.button_inactive} onClick={() => { if (camera.isActive) {setCamport(camera.port); setCamname(camera.id); console.log(camera.port)}}}>
                    <strong>VIDEO ({camera.id})</strong>
                    </button>
                </li>
                ))}
            </ul>
            <br/>
        </div>
    );
}


export default CameraPicker;




