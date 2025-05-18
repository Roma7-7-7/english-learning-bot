// src/components/AuthenticationGuard.tsx
import {useEffect, useState} from 'react';
import {useLocation, useNavigate} from 'react-router-dom';
import {useAppState} from "../context.tsx";
import client, {type Auth} from "../api/client.tsx";

interface AuthenticationGuardProps {
    children: React.ReactNode;
}

export function AuthenticationGuard({children}: AuthenticationGuardProps) {
    const [isLoading, setIsLoading] = useState(true);
    const {dispatch} = useAppState();
    const navigate = useNavigate();
    const location = useLocation();

    useEffect(() => {
        // todo do we need /error here?
        const isPublicRoute = ['/login', '/logout', "/error"].includes(location.pathname);
        if (isPublicRoute) {
            setIsLoading(false);
            return;
        }

        async function checkAuthentication() {
            try {
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
                    dispatch({
                        type: 'LOGIN_SUCCESS', payload: {
                            chatID: a.chat_id,
                        }
                    })
                }).catch(e => {
                    if (e === undefined) {
                        return;
                    }
                    console.error(e);
                    navigate("/error?message=Something went wrong"); // todo create error route
                })
            } finally {
                setIsLoading(false);
            }
        }

        checkAuthentication();
    }, [dispatch, navigate, location.pathname]);

    if (isLoading) {
        // Show loading indicator while checking authentication
        return <div>Loading...</div>;
    }

    return <>{children}</>;
}