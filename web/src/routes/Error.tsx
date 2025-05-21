import {Container, Row} from "react-bootstrap";

export function ErrorPage() {
    const searchParams = new URLSearchParams(window.location.search);
    const message = searchParams.get('message') || "Something went wrong";
    return (
        <Container id="content" className="p-3">
            <Row className="mb-3 align-items-center">
                <h1>Error</h1>
                <p>{message}</p>
            </Row>
        </Container>
    );
}