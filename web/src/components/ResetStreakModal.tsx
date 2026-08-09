import { useState } from "react";
import { Alert, Badge, Button, Modal } from "react-bootstrap";
import client from "../api/client.tsx";

interface ResetStreakModalProps {
  show: boolean;
  word: string;
  translation: string;
  guessedStreak: number;
  streakLimit: number;
  onHide: () => void;
  onSuccess: () => void;
}

export function ResetStreakModal({
  show,
  word,
  translation,
  guessedStreak,
  streakLimit,
  onHide,
  onSuccess,
}: ResetStreakModalProps) {
  const [error, setError] = useState<string>("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [wasShown, setWasShown] = useState(show);

  // Clear the previous attempt's error on the open transition — see
  // https://react.dev/learn/you-might-not-need-an-effect#adjusting-some-state-when-a-prop-changes
  if (show !== wasShown) {
    setWasShown(show);
    if (show) {
      setError("");
    }
  }

  const submit = async (addToBatch: boolean) => {
    setError("");
    setIsSubmitting(true);

    try {
      const response = await client.resetStreak({
        word,
        add_to_batch: addToBatch,
      });

      if (response.ok) {
        onSuccess();
        onHide();
        return;
      }

      const errorData = await response.json().catch(() => ({}));
      setError(errorData.error ?? "Failed to reset streak");
    } catch (e) {
      console.error("Error resetting streak:", e);
      setError("Failed to reset streak. Please try again.");
    } finally {
      setIsSubmitting(false);
    }
  };

  const isLearned = guessedStreak >= streakLimit;

  return (
    <Modal show={show} onHide={onHide} centered backdrop="static" fullscreen="sm-down">
      <Modal.Header closeButton>
        <Modal.Title>Reset Streak</Modal.Title>
      </Modal.Header>
      <Modal.Body className="px-3 px-sm-4">
        <p className="mb-2">
          <span className="fw-semibold">{word}</span>
          {translation && <span className="text-muted"> — {translation}</span>}
        </p>
        <p className="mb-3">
          Current streak:{" "}
          {isLearned ? (
            <Badge bg="success">✓ Learned ({guessedStreak})</Badge>
          ) : (
            <Badge bg="secondary">
              {guessedStreak}/{streakLimit}
            </Badge>
          )}
        </p>
        <p className="text-muted small mb-0">
          Resetting sets the streak back to 0, which also takes the word out of the reviews that
          only cover learned words. Add it to the learning batch to be asked about it again soon;
          otherwise it waits among the words still to be learned until a batch refill picks it up.
        </p>

        {error && (
          <Alert variant="danger" className="mt-3 mb-0">
            {error}
          </Alert>
        )}
      </Modal.Body>
      <Modal.Footer className="d-flex flex-column flex-sm-row gap-2 gap-sm-0">
        <Button
          variant="secondary"
          onClick={onHide}
          disabled={isSubmitting}
          className="w-100 w-sm-auto order-3 order-sm-1"
        >
          Cancel
        </Button>
        <Button
          variant="outline-warning"
          onClick={() => submit(false)}
          disabled={isSubmitting}
          className="w-100 w-sm-auto order-2 order-sm-2"
        >
          Reset only
        </Button>
        <Button
          variant="warning"
          onClick={() => submit(true)}
          disabled={isSubmitting}
          className="w-100 w-sm-auto order-1 order-sm-3"
        >
          {isSubmitting ? "Working..." : "Reset and add to learning batch"}
        </Button>
      </Modal.Footer>
    </Modal>
  );
}
