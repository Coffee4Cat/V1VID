import {React, useState} from "react";
import ModeRedirector from "./ModeRedirector.jsx"
import styles from "./VideoModes.module.css"




function VideoModes() {

    return(
        <div className={styles.wrapper}>
            <div className={styles.title}>
                <br/>
                <h1>Video Modes</h1>
            <div className={styles.description}>
                <p>choose the best solution for your needs</p>
            </div>
            </div>
            <div className={styles.panel}>
                <ul className={styles.modeselect}>
                    <ModeRedirector name="single-camera" link="/singlecamera"/>
                    <ModeRedirector name="dual-camera" link="/dualcamera"/>
                    <ModeRedirector name="every-camera" link="everycamera"/>
                </ul>
            </div>
        </div>
    );
};

export default VideoModes