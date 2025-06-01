import React, { useState } from "react";
import { useNavigate } from "react-router-dom";
import client, { type Status } from "../api/client.tsx";
import { Row, Col, Form, Button, Alert, Container, Spinner } from 'react-bootstrap';

let intervalID = 0;

export function Login() {
    const [errorMessage, setErrorMessage] = useState<string | null>(null);
    const [awaiting, setAwaiting] = useState(false);
    const navigate = useNavigate();

    const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
        event.preventDefault();
        if (errorMessage) {
            setErrorMessage("")
        }
        const form = event.currentTarget;
        const formElements = form.elements as typeof form.elements & {
            chatID: HTMLInputElement;
        };

        client.login(parseInt(formElements.chatID.value)).then((r) => {
            if (!r.ok) {
                if (r.status == 403) {
                    setErrorMessage("Invalid chat ID");
                } else {
                    setErrorMessage("An error occurred while logging in");
                }
                return Promise.reject()
            }

            return r.json() as Promise<Status>;
        }).then(() => {
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
        })
    };

    const handleCancel = () => {
        setErrorMessage("");
        clearInterval(intervalID);
        setAwaiting(false);
    }

    return (
        <Container>
            <Row className="justify-content-center" style={{ marginTop: '100px' }}>
                <Col xs={12} sm={10} md={8} lg={8} xl={6}>
                    <div style={{ backgroundColor: '#f6f6f6', borderRadius: '8px', boxShadow: '0 2px 4px rgba(0,0,0,0.1)', padding: '20px' }}>
                        {awaiting ? (
                            <>
                                <Row className="mb-4">
                                    <Col className="text-center">
                                        <h2 className="mb-4">Approve in telegram chat</h2>
                                        <Spinner animation="border" variant="primary" style={{ width: '100px', height: '100px' }} />
                                    </Col>
                                </Row>
                                <Row>
                                    <Col className="text-center">
                                        <Button variant="danger" onClick={handleCancel}>Cancel</Button>
                                    </Col>
                                </Row>
                            </>
                        ) : (
                            <Form onSubmit={handleSubmit}>
                                <Row className="mb-3">
                                    <Col className="text-center">
                                        <h2>Enter Chat ID</h2>
                                    </Col>
                                </Row>
                                <Row className="mb-4 align-items-center">
                                    <Col xs={12} sm="auto" className="mb-2 mb-sm-0">
                                        <Form.Label htmlFor="chatID" className="mb-0">Chat ID: </Form.Label>
                                    </Col>
                                    <Col xs={12} sm>
                                        <Form.Control
                                            type="text"
                                            id="chatID"
                                            name="chatID"
                                            placeholder="Enter your chat ID"
                                            required
                                        />
                                    </Col>
                                </Row>
                                {errorMessage && (
                                    <Row className="mb-3">
                                        <Col className="text-center">
                                            <Alert variant="danger">
                                                {errorMessage}
                                            </Alert>
                                        </Col>
                                    </Row>
                                )}
                                <Row>
                                    <Col className="text-center">
                                        <Button type="submit" variant="primary" className="px-4">Submit</Button>
                                    </Col>
                                </Row>
                            </Form>
                        )}
                    </div>
                </Col>
            </Row>
        </Container>
    );
}