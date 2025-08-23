import React from "react";
import ControlBlock from "./ControlBlock";
import styles from "./ControlPanel.module.css"
import config from "../../config.js"


function ControlPanel() {
    return(
        <div>
        <h2>Control Panel</h2> click to check if V1VID video server is alright
        <div className={styles.panel}>
            <ul className={styles.blocklist}>
                <ControlBlock text="Server Alive-Status" address={`http://${config.IPADDR}:${config.PORT}/server-status`}/>
            </ul>
        </div>
        </div>
    );
}

export default ControlPanel;