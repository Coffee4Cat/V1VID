import React, { useRef, useState, useEffect } from "react";
import styles from './CameraPicker.module.css';




const CameraPicker = ({title, cameras, setCamport, setCamname}) => {

    const qualityMap = {
        enabled: {
            1: styles.button_active1,
            2: styles.button_active2,
            3: styles.button_active2,
        },
        disabled: {
            1: styles.button_inactive1,
            2: styles.button_inactive2,
            3: styles.button_inactive3,
        },
    };


        // <div className={`${styles.controlblock} ${enable ? qualityMap.enabled[quality] : qualityMap.disabled[quality]}`}></div>
    return (
        <div className={styles.container}>
            <div className={styles.title}>
                <h3>Pick Stream for monitor {title}</h3>
            </div>
            <ul className={styles.camera_list}>
                {cameras.map((camera) => (
                <li key={camera.port}>
                    <button className={camera.isActive ? qualityMap.enabled[camera.quality] : qualityMap.disabled[camera.quality]} onClick={() => { if (camera.isActive) {setCamport(camera.port); setCamname(camera.id); console.log(camera.port)}}}>
                    <strong>({camera.id})</strong>
                    </button>
                </li>
                ))}
            </ul>
            <br/>
        </div>
    );
}


export default CameraPicker;




