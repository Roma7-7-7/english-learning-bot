// src/components/AuthenticationGuard.tsx
import {useEffect, useState} from 'react';
import {useLocation, useNavigate} from 'react-router-dom';
import {useAppState} from "../context.tsx";
import client, {type Auth} from "../api/client.tsx";

interface AuthenticationGuardProps {
    children: React.ReactNode;
}

const PUBLIC_ROUTES = ['/login', '/logout', '/error'] as const;
const AUTH_CHECK_INTERVAL = 5 * 60 * 1000; // 5 minutes

export function AuthenticationGuard({children}: AuthenticationGuardProps) {
    const [isLoading, setIsLoading] = useState(true);
    const {dispatch} = useAppState();
    const navigate = useNavigate();
    const location = useLocation();

    useEffect(() => {
        let authCheckInterval: number;

        const isPublicRoute = PUBLIC_ROUTES.includes(location.pathname as typeof PUBLIC_ROUTES[number]);
        if (isPublicRoute) {
            setIsLoading(false);
            return;
        }

        async function checkAuthentication() {
            try {
                const response = await client.getAuth();
                
                if (response.status === 401) {
                    // Clear any existing auth state
                    dispatch({ type: 'LOGOUT' });
                    navigate("/login", { 
                        state: { 
                            from: location.pathname,
                            message: "Your session has expired. Please log in again." 
                        }
                    });
                    return;
                }

                if (!response.ok) {
                    throw new Error(`HTTP error! status: ${response.status}`);
                }

                const auth = await response.json() as Auth;
                
                if (!auth.chat_id) {
                    throw new Error("Invalid authentication response");
                }

                dispatch({
                    type: 'LOGIN_SUCCESS',
                    payload: {
                        chatID: auth.chat_id,
                    }
                });

                // Set up periodic auth check
                if (!authCheckInterval) {
                    authCheckInterval = window.setInterval(checkAuthentication, AUTH_CHECK_INTERVAL);
                }

            } catch (error) {
                console.error('Authentication error:', error);
                dispatch({ type: 'LOGOUT' });
                navigate("/error", { 
                    state: { 
                        message: "An error occurred while checking authentication status. Please try again later." 
                    }
                });
            } finally {
                setIsLoading(false);
            }
        }

        checkAuthentication();

        // Cleanup interval on unmount
        return () => {
            if (authCheckInterval) {
                window.clearInterval(authCheckInterval);
            }
        };
    }, [dispatch, navigate, location.pathname]);

    if (isLoading) {
        return <div className="loading-spinner">Loading...</div>;
    }

    return <>{children}</>;
}