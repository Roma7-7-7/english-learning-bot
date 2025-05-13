import React, {useState} from "react";
import {useNavigate} from "react-router-dom";
import client, {type Status} from "../api/client.tsx";

let intervalID = 0;

export function Login() {
    const [errorMessage, setErrorMessage] = useState<string | null>(null);
    const [awaiting, setAwaiting] = useState(false);
    const navigate = useNavigate();

    const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
        event.preventDefault();
        setErrorMessage("")
        const form = event.currentTarget;
        await client.login(form.chatID.value)

        intervalID = setInterval(() => {
            client.getStatus().then((r) => {
               return r.json() as Promise<Status>;
            }).then((s) => {
                if (s.authenticated) {
                    clearInterval(intervalID);
                    navigate("/");
                }
            }).catch((error) => {
                clearInterval(intervalID)
                setErrorMessage(error.message);
            });
        }, 1_000)
        setAwaiting(true);
    };

    const handleCancel = () => {
        setErrorMessage("");
        clearInterval(intervalID);
        setAwaiting(false);
    }

    return (
        <div className="row justify-content-center" style={{marginTop: '100px'}}>
            <div className="col-12 col-sm-10 col-md-8 col-lg-8 col-xl-6"
                 style={{backgroundColor: '#f6f6f6', borderRadius: '8px', boxShadow: '0 2px 4px rgba(0,0,0,0.1)', padding: '20px'}}>
                {awaiting && (
                    <>
                        <div className="row mb-4">
                            <div className="col text-center">
                                <h2 className="mb-4">Approve in telegram chat</h2>
                                <svg width="100" height="100" viewBox="0 0 100 100" xmlns="http://www.w3.org/2000/svg">
                                    <circle cx="50" cy="50" r="40" stroke="#5693f5" stroke-width="8" fill="none"
                                            stroke-dasharray="62.8 62.8"
                                            stroke-linecap="round">
                                        <animateTransform attributeName="transform" type="rotate" from="0 50 50" to="360 50 50" dur="1s"
                                                          repeatCount="indefinite"/>
                                    </circle>
                                </svg>
                            </div>
                        </div>
                        <div className="row">
                            <div className="col text-center">
                                <a className="btn btn-danger" onClick={handleCancel}>Cancel</a>
                            </div>
                        </div>
                    </>
                )}
                {!awaiting && (
                    <form id="loginForm" className="align-items-center" onSubmit={handleSubmit}>
                        <div className="row mb-3">
                            <div className="col text-center">
                                <h2>Enter Chat ID</h2>
                            </div>
                        </div>
                        <div className="row mb-4 align-items-center">
                            <div className="col-12 col-sm-auto mb-2 mb-sm-0">
                                <label htmlFor="chatID" className="col-form-label mb-0">Chat ID: </label>
                            </div>
                            <div className="col-12 col-sm">
                                <input type="text" className="form-control" id="chatID" name="chatID" placeholder="Enter your chat ID"
                                       required/>
                            </div>
                        </div>
                        {errorMessage && (
                            <div className="row mb-3">
                                <div className="col text-center">
                                    <div className="alert alert-danger" role="alert">
                                        {errorMessage}
                                    </div>
                                </div>
                            </div>
                        )}
                        <div className="row">
                            <div className="col text-center">
                                <button type="submit" className="btn btn-primary px-4">Submit</button>
                            </div>
                        </div>
                    </form>
                )}
            </div>
        </div>
    )
}