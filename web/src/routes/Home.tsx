import {useEffect} from "react";
import client from "../api/client.tsx";
import {useNavigate} from "react-router-dom";

export function Home() {
    const navigate = useNavigate();

    if (window.location.pathname !== '/login') {
        useEffect(() => {
            client.getStatus().then((r) => {
                if (r.status == 401) {
                    return navigate("/login")
                }

                if (r.status != 200) {
                    throw new Error("Unexpected status code: " + r.status);
                }
            }).catch(e => {
                console.error(e);
                navigate("/error?message=Something went wrong"); // todo create error route
            })
        }, []);
    }

    return (
        <>
            {}
        </>
    )
}
