-- Adds the FIFO admission queue that backs the learning batch's new hard cap.
--
-- BOT_LEARNING_BATCH_SIZE is now enforced as a hard cap on learning_batches. A word that wants back
-- in (a miss, a deliberate reset, a conflict resolution, or a brand-new word) while the batch is
-- already full is appended here instead of being dropped or pushing the batch over size, and is
-- drained oldest-first by RefillLearningBatch as room frees up.
--
-- Apply once to an existing database:
--     sqlite3 data/db.sqlite < schema/migrations/002_learning_batch_queue.sql
--
-- New databases created from schema/schema_sqlite.sql already include this.
--
-- No backfill needed: words that predate this migration and are not currently batched are picked up
-- by RefillLearningBatch's existing random-fill fallback the next time there is room, exactly as they
-- were before this table existed. That fallback is not a transitional shim — it remains the
-- permanent path for words nobody has explicitly asked to re-admit.

CREATE TABLE learning_batch_queue
(
    chat_id    INTEGER NOT NULL,
    word       TEXT    NOT NULL,
    queued_seq INTEGER NOT NULL,

    PRIMARY KEY (chat_id, word),
    FOREIGN KEY (chat_id, word)
    REFERENCES word_translations (chat_id, word)
    ON DELETE CASCADE
    ON UPDATE CASCADE
);

CREATE INDEX idx_learning_batch_queue_chat_id_seq
    ON learning_batch_queue (chat_id, queued_seq);
