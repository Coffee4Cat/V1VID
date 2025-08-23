import React, {useState} from "react";
import styles from "./CameraActivator.module.css"


function CameraActivator({text, address, camera_id, entry_status, entry_quality}) {
    const [enable, setEnable] = useState(entry_status);
    const [quality, setQuality] = useState(entry_quality);


    const qualityMap = {
        enabled: {
            1: styles.enabledquality1,
            2: styles.enabledquality2,
            3: styles.enabledquality3,
        },
        disabled: {
            1: styles.disabledquality1,
            2: styles.disabledquality2,
            3: styles.disabledquality3,
        },
        title: {
            1: "INDOR",
            2: "CLOUDY",
            3: "SUNNY",
        },
    };
    
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

    const handleIndorQuality = async () => {
        if (!enable) {
            try {
                let addr;
                addr = address + "/indorquality/" + camera_id;
                const response = await fetch(addr, {method: "POST"});
                const data = await response.json();
                setQuality(1);
            } catch (error) {}
        }
    };
    
    const handleCloudyQuality = async () => {
        if (!enable) {
            try {
                let addr;
                addr = address + "/cloudyquality/" + camera_id;
                const response = await fetch(addr, {method: "POST"});
                const data = await response.json();
                setQuality(2);
            } catch (error) {}
        }
    };

    const handleSunnyQuality = async () => {
        if (!enable) {
            try {
                let addr;
                addr = address + "/sunnyquality/" + camera_id;
                const response = await fetch(addr, {method: "POST"});
                const data = await response.json();
                setQuality(3);
            } catch (error) {}
        }
    };
    
    
    
    return (
        <div className={`${styles.controlblock} ${enable ? qualityMap.enabled[quality] : qualityMap.disabled[quality]}`}>
            <p>{text}</p>
            <button className={styles.button} onClick={handleClick}>{enable ? "TURN OFF" : "TURN ON"}</button>
            <p>MODE {qualityMap.title[quality]}</p>
            <ul className={styles.modelist}>
                <li><button className={styles.qbutton1} onClick={handleIndorQuality}>INDOR</button></li>
                <li><button className={styles.qbutton2} onClick={handleCloudyQuality}>CLOUDY</button></li>
                <li><button className={styles.qbutton3} onClick={handleSunnyQuality}>SUNNY</button></li>
            </ul>
        </div>
    );
}

export default CameraActivator;
