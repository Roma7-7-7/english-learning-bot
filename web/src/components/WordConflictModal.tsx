import { useState } from "react";
import { Alert, Badge, Button, Form, Modal } from "react-bootstrap";
import client, { type ConflictResolution, type Word } from "../api/client.tsx";

interface WordConflictModalProps {
  show: boolean;
  /** The entry already stored for this word. */
  existing: Word;
  /** What the user just tried to add. */
  incoming: Word;
  streakLimit: number;
  onHide: () => void;
  onSuccess: () => void;
}

type TranslationChoice = "existing" | "incoming";

export function WordConflictModal({
  show,
  existing,
  incoming,
  streakLimit,
  onHide,
  onSuccess,
}: WordConflictModalProps) {
  const [choice, setChoice] = useState<TranslationChoice>("incoming");
  const [error, setError] = useState<string>("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [wasShown, setWasShown] = useState(show);

  // Reset the previous conflict's answer on the open transition — see
  // https://react.dev/learn/you-might-not-need-an-effect#adjusting-some-state-when-a-prop-changes
  if (show !== wasShown) {
    setWasShown(show);
    if (show) {
      setChoice("incoming");
      setError("");
    }
  }

  const differs =
    existing.translation !== incoming.translation ||
    (existing.description || "") !== (incoming.description || "");

  const submit = async (onConflict: ConflictResolution) => {
    setError("");
    setIsSubmitting(true);

    // Whichever text the user picked is simply what we send; the server writes it verbatim.
    const chosen = !differs || choice === "incoming" ? incoming : existing;

    try {
      const response = await client.createWord({
        word: existing.word,
        translation: chosen.translation,
        description: chosen.description ?? "",
        on_conflict: onConflict,
      });

      if (response.ok) {
        onSuccess();
        onHide();
        return;
      }

      const errorData = await response.json().catch(() => ({}));
      setError(errorData.error ?? "Failed to resolve the conflict");
    } catch (e) {
      console.error("Error resolving word conflict:", e);
      setError("Failed to resolve the conflict. Please try again.");
    } finally {
      setIsSubmitting(false);
    }
  };

  const streak = existing.guessed_streak || 0;
  const isLearned = streak >= streakLimit;

  return (
    <Modal show={show} onHide={onHide} centered backdrop="static" fullscreen="sm-down">
      <Modal.Header closeButton>
        <Modal.Title>Word already exists</Modal.Title>
      </Modal.Header>
      <Modal.Body className="px-3 px-sm-4">
        <p className="mb-2">
          <span className="fw-semibold">{existing.word}</span> is already in your vocabulary
          {" — "}
          {isLearned ? (
            <Badge bg="success">✓ Learned ({streak})</Badge>
          ) : (
            <Badge bg="secondary">
              {streak}/{streakLimit}
            </Badge>
          )}
        </p>

        {differs ? (
          <Form.Group className="mb-3">
            <Form.Label className="fw-semibold">Which translation should be kept?</Form.Label>
            <Form.Check
              type="radio"
              id="conflict-keep-existing"
              name="conflict-translation"
              checked={choice === "existing"}
              onChange={() => setChoice("existing")}
              label={
                <span>
                  Keep existing: <span className="fw-semibold">{existing.translation}</span>
                  {existing.description && (
                    <span className="text-muted small d-block">{existing.description}</span>
                  )}
                </span>
              }
            />
            <Form.Check
              type="radio"
              id="conflict-use-new"
              name="conflict-translation"
              checked={choice === "incoming"}
              onChange={() => setChoice("incoming")}
              label={
                <span>
                  Use new: <span className="fw-semibold">{incoming.translation}</span>
                  {incoming.description && (
                    <span className="text-muted small d-block">{incoming.description}</span>
                  )}
                </span>
              }
            />
          </Form.Group>
        ) : (
          <p className="text-muted mb-3">
            The translation you entered matches the one already stored.
          </p>
        )}

        <p className="text-muted small mb-0">
          If you were adding this word again because you had forgotten it, reset the streak. Add it
          to the learning batch as well to be asked about it regularly rather than only during
          occasional reviews.
        </p>

        {error && (
          <Alert variant="danger" className="mt-3 mb-0">
            {error}
          </Alert>
        )}
      </Modal.Body>
      <Modal.Footer className="d-flex flex-column gap-2">
        <Button
          variant="warning"
          onClick={() => submit("reset_and_batch")}
          disabled={isSubmitting}
          className="w-100"
        >
          Reset and add to learning batch
        </Button>
        <Button
          variant="outline-warning"
          onClick={() => submit("reset_only")}
          disabled={isSubmitting}
          className="w-100"
        >
          Reset only
        </Button>
        <Button
          variant="outline-primary"
          onClick={() => submit("update_only")}
          disabled={isSubmitting}
          className="w-100"
        >
          Update translation only (keep streak)
        </Button>
        <Button variant="secondary" onClick={onHide} disabled={isSubmitting} className="w-100">
          Cancel
        </Button>
      </Modal.Footer>
    </Modal>
  );
}
