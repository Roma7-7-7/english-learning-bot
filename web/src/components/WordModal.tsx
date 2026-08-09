import { useState, useEffect } from "react";
import { Modal, Button, Form, Alert } from "react-bootstrap";
import client, { type Word, type WordConflictResponse } from "../api/client.tsx";
import { WordConflictModal } from "./WordConflictModal.tsx";

export type WordModalAction = 'add' | 'edit';

interface WordModalProps {
    show: boolean;
    action: WordModalAction;
    word?: string;
    translation: string;
    description?: string;
    streakLimit: number;
    onHide: () => void;
    onSuccess: () => void;
}

interface ConflictState {
    existing: Word;
    incoming: Word;
}

export function WordModal({
                              show,
                              action,
                              word = "",
                              translation = "",
                              description = "",
                              streakLimit,
                              onHide,
                              onSuccess
                          }: WordModalProps) {
    const [wordInput, setWordInput] = useState(word);
    const [newWordInput, setNewWordInput] = useState(word);
    const [translationInput, setTranslationInput] = useState(translation);
    const [descriptionInput, setDescriptionInput] = useState(description);
    const [error, setError] = useState<string>("");
    const [isSubmitting, setIsSubmitting] = useState(false);
    const [isMobile, setIsMobile] = useState(window.innerWidth < 576);
    const [wasShown, setWasShown] = useState(show);
    const [conflict, setConflict] = useState<ConflictState | null>(null);

    // Reset form on open transition by adjusting state during render — see
    // https://react.dev/learn/you-might-not-need-an-effect#adjusting-some-state-when-a-prop-changes
    if (show !== wasShown) {
        setWasShown(show);
        if (show) {
            setWordInput(word);
            setNewWordInput(word);
            setTranslationInput(translation);
            setDescriptionInput(description);
            setError("");
            setConflict(null);
        }
    }

    // Handle window resize for responsive design
    useEffect(() => {
        const handleResize = () => {
            setIsMobile(window.innerWidth < 576);
        };

        window.addEventListener('resize', handleResize);
        return () => window.removeEventListener('resize', handleResize);
    }, []);

    // Focus the appropriate input after the modal is shown
    useEffect(() => {
        if (!show) {
            return;
        }
        const focusElement = action === 'add' ? "word-input" : "new-word-input";
        const element = document.getElementById(focusElement);
        if (element) {
            (element as HTMLInputElement).focus();
        }
    }, [show, action]);

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
                if (action === 'edit') {
                    client.markToReview({
                        word: newWordInput,
                        to_review: false,
                    });
                }
                onSuccess();
                onHide();
                return;
            }

            // The word is already in the vocabulary: ask what to do with its existing progress
            // rather than silently overwriting it.
            if (response.status === 409) {
                const body: WordConflictResponse = await response.json();
                setConflict({
                    existing: body.existing,
                    incoming: {
                        word: wordInput,
                        translation: translationInput,
                        description: descriptionInput,
                    },
                });
                return;
            }

            const errorData = await response.json().catch(() => ({}));
            setError(errorData.error ?? `Failed to ${action} word`);
        } catch (err) {
            console.error(`Error ${action === 'add' ? 'adding' : 'updating'} word:`, err);
            setError(`Failed to ${action} word. Please try again.`);
        } finally {
            setIsSubmitting(false);
        }
    };

    const title = action === 'add' ? 'Add New Word' : 'Edit Word';
    const submitButtonText = action === 'add' ? 'Add Word' : 'Save Changes';

    if (conflict) {
        return (
            <WordConflictModal
                show={show}
                existing={conflict.existing}
                incoming={conflict.incoming}
                streakLimit={streakLimit}
                // Backing out of the question returns to the form: closing the dialog would drop
                // everything the user typed, and re-opening it starts from blank inputs.
                onBack={() => setConflict(null)}
                onHide={onHide}
                onSuccess={onSuccess}
            />
        );
    }

    return (
        <Modal
            show={show}
            onHide={onHide}
            centered
            backdrop="static"
            fullscreen="sm-down"
        >
            <Modal.Header closeButton>
                <Modal.Title>{title}</Modal.Title>
            </Modal.Header>
            <Form onSubmit={handleSubmit}>
                <Modal.Body className="px-3 px-sm-4">
                    {action === 'edit' && (
                        <Form.Group className="mb-3">
                            <Form.Label className="fw-semibold">Original Word</Form.Label>
                            <Form.Control
                                type="text"
                                value={wordInput}
                                readOnly
                                disabled
                                className="bg-light"
                            />
                        </Form.Group>
                    )}

                    {action === 'edit' ? (
                        <Form.Group className="mb-3">
                            <Form.Label className="fw-semibold">New Word</Form.Label>
                            <Form.Control
                                id="new-word-input"
                                type="text"
                                value={newWordInput}
                                onChange={(e) => setNewWordInput(e.target.value)}
                                placeholder="Enter new word"
                                required
                                size={isMobile ? "lg" : undefined}
                            />
                        </Form.Group>
                    ) : (
                        <Form.Group className="mb-3">
                            <Form.Label className="fw-semibold">Word</Form.Label>
                            <Form.Control
                                id="word-input"
                                type="text"
                                value={wordInput}
                                onChange={(e) => setWordInput(e.target.value)}
                                placeholder="Enter word"
                                required
                                size={isMobile ? "lg" : undefined}
                            />
                        </Form.Group>
                    )}

                    <Form.Group className="mb-3">
                        <Form.Label className="fw-semibold">Translation</Form.Label>
                        <Form.Control
                            type="text"
                            value={translationInput}
                            onChange={(e) => setTranslationInput(e.target.value)}
                            placeholder="Enter translation"
                            required
                            size={isMobile ? "lg" : undefined}
                        />
                    </Form.Group>

                    <Form.Group className="mb-3">
                        <Form.Label className="fw-semibold">Description</Form.Label>
                        <Form.Control
                            as="textarea"
                            rows={isMobile ? 2 : 3}
                            value={descriptionInput}
                            onChange={(e) => setDescriptionInput(e.target.value)}
                            placeholder="Enter description"
                            style={{ minHeight: '60px' }}
                            size={isMobile ? "lg" : undefined}
                        />
                    </Form.Group>

                    {error && (
                        <Alert variant="danger">{error}</Alert>
                    )}
                </Modal.Body>
                <Modal.Footer className="d-flex flex-column flex-sm-row gap-2 gap-sm-0">
                    <Button 
                        variant="secondary" 
                        onClick={onHide} 
                        disabled={isSubmitting}
                        className="w-100 w-sm-auto order-2 order-sm-1"
                    >
                        Cancel
                    </Button>
                    <Button
                        variant="primary"
                        type="submit"
                        disabled={isSubmitting}
                        className="w-100 w-sm-auto order-1 order-sm-2"
                    >
                        {isSubmitting ? 'Saving...' : submitButtonText}
                    </Button>
                </Modal.Footer>
            </Form>
        </Modal>
    );
}