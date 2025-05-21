import { useAppState } from "../context.tsx";
import client from "../api/client.tsx";
import {Link, useNavigate} from "react-router-dom";
import {Navbar as BSNavbar, Container, Nav, Button} from 'react-bootstrap';

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

    return (
        <BSNavbar bg="dark" variant="dark" expand="lg" className="mb-3" style={{borderRadius: "0 0 10px 10px"}}>
            <Container fluid>
                <Link to="/" className="navbar-brand">Home</Link>
                <BSNavbar.Toggle aria-controls="navbarScroll" />
                <BSNavbar.Collapse id="navbarScroll">
                    <Nav className="me-auto my-2 my-lg-0" />
                    <span style={{margin: '0 15px'}}>
                        <span style={{color: 'forestgreen'}}>{state.stats?.learned}</span>
                        <span style={{color: 'whitesmoke'}}> / </span>
                        <span style={{color: 'indianred'}}>{state.stats?.total}</span>
                    </span>
                    {state.user && <Button variant="outline-danger" onClick={handleLogout}>Log out</Button>}
                </BSNavbar.Collapse>
            </Container>
        </BSNavbar>
    );
}