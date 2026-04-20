import {React, useState} from "react";
import { Link }  from "react-router-dom";
import styles from "./ModeRedirector.module.css"


function ModeRedirector({name, link}) {




    return(
        <Link to={link} className={styles.wrapper}>
            <div>
                <p>{name}</p>
            </div>
        </Link>
    );
};


export default ModeRedirector;



