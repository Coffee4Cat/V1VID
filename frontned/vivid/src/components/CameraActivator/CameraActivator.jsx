import React, {useState} from "react";
import styles from "./CameraActivator.module.css"


function CameraActivator({text, address, camera_id, entry_status}) {
    const [enable, setEnable] = useState(entry_status);
    
    const handleClick = async () => {
        try {
            let addr;
            if (enable) {
                addr = address + "/stop/" + camera_id;
            } else {
                addr = address + "/start/" + camera_id;
            }
            const response = await fetch(addr, {method: "POST"});
            const data = await response.json();
            const status = data.status
            if (status) {
                setEnable(true);
            }
            else {
                setEnable(false);
            }
        } catch (error) {
            setEnable(false);

        }
    };
    
    return (
        <div className={`${styles.controlblock} ${enable ? styles.enabled : ""}`}>
            <p>{text}</p>
            <button className={styles.button} onClick={handleClick}></button>
        </div>
    );
}

export default CameraActivator;
