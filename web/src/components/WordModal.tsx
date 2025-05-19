import { useState, useEffect } from "react";
import { Modal, Button, Form, Alert } from "react-bootstrap";
import client from "../api/client.tsx";

export type WordModalAction = 'add' | 'edit';

interface WordModalProps {
    show: boolean;
    action: WordModalAction;
    word?: string;
    translation: string;
    description?: string;
    onHide: () => void;
    onSuccess: () => void;
}

export function WordModal({
                              show,
                              action,
                              word = "",
                              translation = "",
                              description = "",
                              onHide,
                              onSuccess
                          }: WordModalProps) {
    const [wordInput, setWordInput] = useState(word);
    const [newWordInput, setNewWordInput] = useState(word);
    const [translationInput, setTranslationInput] = useState(translation);
    const [descriptionInput, setDescriptionInput] = useState(description);
    const [error, setError] = useState<string>("");
    const [isSubmitting, setIsSubmitting] = useState(false);

    // Reset form when modal opens with new props
    useEffect(() => {
        if (show) {
            setWordInput(word);
            setNewWordInput(word);
            setTranslationInput(translation);
            setDescriptionInput(description);
            setError("");
            const focusElement = action === 'add' ? "word-input" : "new-word-input";
            const element = document.getElementById(focusElement);
            if (element) {
                (element as HTMLInputElement).focus();
            }
        }
    }, [show, word, translation, description, action]);

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setError("");
        setIsSubmitting(true);

        try {
            let response;

            if (!confirm("Are you sure you want to proceed?")) {
                return;
            }
            if (action === 'add') {
                response = await client.createWord({
                    word: wordInput,
                    translation: translationInput,
                    description: descriptionInput,
                });
            } else {
                response = await client.updateWord({
                    word: wordInput,
                    new_word: newWordInput,
                    translation: translationInput,
                    description: descriptionInput,
                });
            }

            if (response.status >= 200 && response.status < 300) {
                onSuccess();
                onHide();
            } else {
                const errorData = await response.json();
                setError(errorData.message || `Failed to ${action} word`);
            }
        } catch (err) {
            console.error(`Error ${action === 'add' ? 'adding' : 'updating'} word:`, err);
            setError(`Failed to ${action} word. Please try again.`);
        } finally {
            setIsSubmitting(false);
        }
    };

    const title = action === 'add' ? 'Add New Word' : 'Edit Word';
    const submitButtonText = action === 'add' ? 'Add Word' : 'Save Changes';

    return (
        <Modal show={show} onHide={onHide} centered backdrop="static">
            <Modal.Header closeButton>
                <Modal.Title>{title}</Modal.Title>
            </Modal.Header>
            <Form onSubmit={handleSubmit}>
                <Modal.Body>
                    {action === 'edit' && (
                        <Form.Group className="mb-3">
                            <Form.Label>Original Word</Form.Label>
                            <Form.Control
                                type="text"
                                value={wordInput}
                                readOnly
                                disabled
                            />
                        </Form.Group>
                    )}

                    {action === 'edit' ? (
                        <Form.Group className="mb-3">
                            <Form.Label>New Word</Form.Label>
                            <Form.Control
                                id="new-word-input"
                                type="text"
                                value={newWordInput}
                                onChange={(e) => setNewWordInput(e.target.value)}
                                placeholder="Enter new word"
                                required
                            />
                        </Form.Group>
                    ) : (
                        <Form.Group className="mb-3">
                            <Form.Label>Word</Form.Label>
                            <Form.Control
                                id="word-input"
                                type="text"
                                value={wordInput}
                                onChange={(e) => setWordInput(e.target.value)}
                                placeholder="Enter word"
                                required
                            />
                        </Form.Group>
                    )}

                    <Form.Group className="mb-3">
                        <Form.Label>Translation</Form.Label>
                        <Form.Control
                            type="text"
                            value={translationInput}
                            onChange={(e) => setTranslationInput(e.target.value)}
                            placeholder="Enter translation"
                            required
                        />
                    </Form.Group>

                    <Form.Group className="mb-3">
                        <Form.Label>Description</Form.Label>
                        <Form.Control
                            as="textarea"
                            rows={3}
                            value={descriptionInput}
                            onChange={(e) => setDescriptionInput(e.target.value)}
                            placeholder="Enter description"
                        />
                    </Form.Group>

                    {error && (
                        <Alert variant="danger">{error}</Alert>
                    )}
                </Modal.Body>
                <Modal.Footer>
                    <Button variant="secondary" onClick={onHide} disabled={isSubmitting}>
                        Cancel
                    </Button>
                    <Button
                        variant="primary"
                        type="submit"
                        disabled={isSubmitting}
                    >
                        {isSubmitting ? 'Saving...' : submitButtonText}
                    </Button>
                </Modal.Footer>
            </Form>
        </Modal>
    );
}