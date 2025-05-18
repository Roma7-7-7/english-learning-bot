import {useAppState} from "../context.tsx";
import {useEffect, useState} from "react";
import client, {type Stats} from "../api/client.tsx";

export function Navbar() {
    const {state, dispatch} = useAppState();
    const [stats, setStats] = useState<Stats>({
        learned: 0,
        total: 0,
    } as Stats);

    useEffect(() => {
        if (state.user == null) {
            return
        }
        client.getStats().then(
            (r) => {
                if (r.status >= 300) {
                    throw new Error("Unexpected status code: " + r.status);
                }
                return r.json();
            }
        ).then((s) => {
            setStats(s);
        }).catch((e) => {
            if (e === undefined) {
                return;
            }
            console.error(e);
        });
    }, [state.user])

    function handleLogout() {
        client.logout().then((r) => {
            if (r.status >= 300) {
                throw new Error("Unexpected status code: " + r.status);
            }
            return r.json();
        }).then(() => {
            dispatch({type: 'LOGOUT'});
            window.location.href = "/login";
        }).catch((e) => {
            if (e === undefined) {
                return;
            }
            console.error(e);
        });
    }

    if (!state.user) {
        return <></>
    }

    return <nav className="navbar bg-dark navbar-expand-lg bg-body-tertiary mb-3" data-bs-theme="dark"
                style={{borderRadius: "0 0 10px 10px"}}>
        <div className="container-fluid">
            <a className="navbar-brand" href="/words">Home</a>
            <button className="navbar-toggler" type="button" data-bs-toggle="collapse" data-bs-target="#navbarSupportedContent"
                    aria-controls="navbarSupportedContent" aria-expanded="false" aria-label="Toggle navigation">
                <span className="navbar-toggler-icon"></span>
            </button>
            <div className="collapse navbar-collapse" id="navbarSupportedContent">
                <ul className="navbar-nav me-auto mb-2 mb-lg-0">
                    <li className="nav-item">
                    </li>
                </ul>
                <span style={{margin: '0 15px'}}><span style={{color: 'forestgreen'}}>{stats.learned}</span> <span
                    style={{color: 'whitesmoke'}}> / </span> <span style={{color: 'indianred'}}> {stats.total}</span></span>
                <a className="btn btn-outline-danger" onClick={handleLogout}>Log out</a>
            </div>
        </div>
    </nav>
}