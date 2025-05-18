import {useEffect} from "react";
import client, {type Auth} from "../api/client.tsx";
import {useNavigate} from "react-router-dom";
import {useAppState} from "../context.tsx";

export function Home() {
    const { state, dispatch } = useAppState();
    const navigate = useNavigate();

    useEffect(() => {
        client.getAuth().then((r) => {
            if (r.status == 401) {
                navigate("/login")
                return Promise.reject()
            }

            if (r.status >= 300) {
                throw new Error("Unexpected status code: " + r.status);
            }

            return r.json() as Promise<Auth>;
        }).then(a => {
            dispatch({type: 'LOGIN_SUCCESS', payload: {
                chatID: a.chat_id,
            }})
        }).catch(e => {
            if (e === undefined) {
                return;
            }
            console.error(e);
            navigate("/error?message=Something went wrong"); // todo create error route
        })
    }, []);

    return (
        <>
            {state.user && (
                <>
                    {state.user.chatID}
                </>
            )}
        </>
    )
}
