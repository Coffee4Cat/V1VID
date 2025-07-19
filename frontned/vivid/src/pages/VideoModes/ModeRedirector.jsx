import {React, useState} from "react";
import { Link }  from "react-router-dom";
import styles from "./ModeRedirector.module.css"


function ModeRedirector({name, link}) {




    return(
        <div className={styles.wrapper}>
            <p>{name}</p>
            <Link to={link}>[CLICK ME]</Link>
        </div>
    );
};


export default ModeRedirector;



