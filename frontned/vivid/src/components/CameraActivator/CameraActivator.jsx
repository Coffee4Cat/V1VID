import React, {useState} from "react";
import styles from "./CameraActivator.module.css"


function CameraActivator({text, address, camera_id, entry_status, entry_quality}) {
    const [enable, setEnable] = useState(entry_status);
    const [quality, setQuality] = useState(entry_quality);
    
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

    const handleGoodQuality = async () => {
        if (!enable) {
            try {
                let addr;
                addr = address + "/goodquality/" + camera_id;
                const response = await fetch(addr, {method: "POST"});
                const data = await response.json();
                setQuality(true);
            } catch (error) {}
        }
    };
    
    const handleBadQuality = async () => {
        if (!enable) {
            try {
                let addr;
                addr = address + "/badquality/" + camera_id;
                const response = await fetch(addr, {method: "POST"});
                const data = await response.json();
                setQuality(false);
            } catch (error) {}
        }
    };
    
    
    return (
        <div className={`${styles.controlblock} ${enable ? (quality ? styles.enabledquality1 : styles.enabledquality2) : (quality ? styles.disabledquality1 : styles.disabledquality2)}`}>
            <p>{text}</p>
            <button className={styles.button} onClick={handleClick}>{enable ? "TURN OFF" : "TURN ON"}</button>
            <p>MODE {quality ? "PREMIUM " : "FAST"}</p>
            <ul className={styles.modelist}>
                <li><button className={styles.qbutton1} onClick={handleGoodQuality}>Premium</button></li>
                <li><button className={styles.qbutton2} onClick={handleBadQuality}>Fast</button></li>
            </ul>
        </div>
    );
}

export default CameraActivator;
