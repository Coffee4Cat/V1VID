import React, {useState} from "react";
import styles from "./ControlBlock.module.css"


function ControlBlock({text, address}) {
    const [enable, setEnable] = useState(false);
    
    const handleClick = async () => {
        try {
            const response = await fetch(address, {method: "POST"});
            const data = await response.json();
            const status = data.status
            if (status) {
                setEnable(true);
            }
        } catch (error) {
            console.warn("[ControlBlock][REQUEST ERROR]: ", error);
            setEnable(false);

        }
    };
    
    return (
        <div className={`${styles.controlblock} ${enable ? styles.enabled : ""}`}>
            <p>{text}</p>
            <button className={styles.button} onClick={handleClick}>Prompt Server</button>
        </div>
    );
}

export default ControlBlock;


