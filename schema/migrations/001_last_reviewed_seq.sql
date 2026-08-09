-- Adds review rotation tracking for already-learned words.
--
-- Apply once to an existing database:
--     sqlite3 data/db.sqlite < schema/migrations/001_last_reviewed_seq.sql
--
-- New databases created from schema/schema_sqlite.sql already include this.
--
-- Existing rows get NULL, meaning "never reviewed", so the first pass of reviews covers the whole
-- learned vocabulary before repeating anything.

ALTER TABLE word_translations ADD COLUMN last_reviewed_seq INTEGER;

CREATE INDEX IF NOT EXISTS idx_word_translations_review
    ON word_translations (chat_id, guessed_streak, last_reviewed_seq);
