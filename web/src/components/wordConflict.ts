import type { ConflictResolution, Word } from "../api/client.tsx";

/**
 * What re-adding an existing word could still change about it.
 *
 * Only these decide what the user is asked. A resolution that would change nothing is not worth a
 * button: resetting a streak that is already zero, or batching a word that is already batched, does
 * nothing the user could notice.
 */
export interface ConflictChoices {
  /** There is progress a reset would discard. */
  canReset: boolean;
  /** The word is outside the learning batch, so adding it back changes when it comes up. */
  canBatch: boolean;
  /** The text differs from what is stored, so there is something to pick between. */
  textDiffers: boolean;
}

export function conflictChoices(existing: Word, incoming: Word): ConflictChoices {
  return {
    canReset: (existing.guessed_streak || 0) > 0,
    canBatch: !existing.in_batch,
    textDiffers:
      existing.translation !== incoming.translation ||
      (existing.description || "") !== (incoming.description || ""),
  };
}

/**
 * True when re-adding the word is a no-op: same text, nothing to reset, already batched. The caller
 * should just apply it instead of opening a dialog whose every button changes nothing.
 */
export function isTrivialConflict(existing: Word, incoming: Word): boolean {
  const { canReset, canBatch, textDiffers } = conflictChoices(existing, incoming);
  return !canReset && !canBatch && !textDiffers;
}

export interface ResolutionAction {
  resolution: ConflictResolution;
  label: string;
  variant: string;
}

/**
 * The buttons to offer, one per outcome that is actually distinct.
 *
 * `reset_and_batch` doubles as plain "add to the batch" once the streak is already zero, and
 * `reset_only` as plain "reset" once the word is already batched — in both cases the other half of
 * the resolution is a no-op, so it needs no button of its own.
 */
export function resolutionActions({ canReset, canBatch }: ConflictChoices): ResolutionAction[] {
  if (canReset && canBatch) {
    return [
      {
        resolution: "reset_and_batch",
        label: "Reset streak and add to learning batch",
        variant: "warning",
      },
      { resolution: "reset_only", label: "Reset streak only", variant: "outline-warning" },
      { resolution: "update_only", label: "Keep the current streak", variant: "outline-primary" },
    ];
  }
  if (canReset) {
    return [
      { resolution: "reset_only", label: "Reset streak", variant: "warning" },
      { resolution: "update_only", label: "Keep the current streak", variant: "outline-primary" },
    ];
  }
  if (canBatch) {
    return [
      { resolution: "reset_and_batch", label: "Add to learning batch", variant: "warning" },
      { resolution: "update_only", label: "Leave it out of the batch", variant: "outline-primary" },
    ];
  }
  // Nothing about the progress can change, so the only thing being saved is the new text.
  return [{ resolution: "update_only", label: "Save translation", variant: "primary" }];
}

export function conflictExplanation({ canReset, canBatch }: ConflictChoices): string {
  if (canReset && canBatch) {
    return (
      "Reset the streak if you are adding this word again because you had forgotten it. A reset " +
      "word is no longer picked for reviews, so add it to the learning batch as well to be asked " +
      "about it again soon; otherwise it waits until a batch refill picks it up."
    );
  }
  if (canReset) {
    return (
      "Reset the streak if you are adding this word again because you had forgotten it. It is " +
      "already in the learning batch, so it will keep coming up either way."
    );
  }
  if (canBatch) {
    return (
      "Its streak is already zero, so there is nothing to reset. Add it to the learning batch to " +
      "be asked about it again soon; otherwise it waits until a batch refill picks it up."
    );
  }
  return (
    "Its streak is already zero and it is already in the learning batch, so only the translation " +
    "changes."
  );
}
