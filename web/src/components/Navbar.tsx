import {useAppState} from "../context.tsx";
import client from "../api/client.tsx";
import {useNavigate} from "react-router-dom";

export function Navbar() {
    const { state, dispatch } = useAppState();
    const navigate = useNavigate();

    function handleLogout() {
        client.logout().then((r) => {
            if (r.status >= 300) {
                throw new Error("Unexpected status code: " + r.status);
            }
            return r.json();
        }).then(() => {
            dispatch({type: 'LOGOUT'});
            navigate("/login");
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
            <a className="navbar-brand" href="/">Home</a>
            <button className="navbar-toggler" type="button" data-bs-toggle="collapse" data-bs-target="#navbarSupportedContent"
                    aria-controls="navbarSupportedContent" aria-expanded="false" aria-label="Toggle navigation">
                <span className="navbar-toggler-icon"></span>
            </button>
            <div className="collapse navbar-collapse" id="navbarSupportedContent">
                <ul className="navbar-nav me-auto mb-2 mb-lg-0">
                    <li className="nav-item">
                    </li>
                </ul>
                <span style={{margin: '0 15px'}}><span style={{color: 'forestgreen'}}>{state.stats?.learned }</span> <span
                    style={{color: 'whitesmoke'}}> / </span> <span style={{color: 'indianred'}}> {state.stats?.total}</span></span>
                <a className="btn btn-outline-danger" onClick={handleLogout}>Log out</a>
            </div>
        </div>
    </nav>
}