import React from "react";
import ControlBlock from "./ControlBlock";
import styles from "./ControlPanel.module.css"


function ControlPanel() {
    return(
        <div>
        <h2>Control Panel</h2> click to check if V1VID video server is alright
        <div className={styles.panel}>
            <ul className={styles.blocklist}>
                <ControlBlock text="Server Alive-Status" address="http://localhost:8080/server-status"/>
                <ControlBlock text="Camera Alive-Status" address="http://localhost:8080/camera-status"/>
                <ControlBlock text="Camera Stream" address="http://localhost:8080/stream"/> 
                {/* some ControlBlocks are only for test purposes */}
            </ul>
        </div>
        </div>
    );
}

export default ControlPanel;