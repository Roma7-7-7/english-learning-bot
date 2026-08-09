CREATE TABLE word_translations
(
    chat_id        INTEGER     NOT NULL,
    word           TEXT        NOT NULL,
    translation    TEXT        NOT NULL,
    description    TEXT,
    guessed_streak INTEGER     NOT NULL DEFAULT 0,
    to_review      INTEGER     NOT NULL DEFAULT 0,
    -- Monotonic per-chat cursor, bumped whenever a learned word is sent out for review. Reviews are
    -- served in ascending order, so they rotate through the whole learned vocabulary before
    -- repeating. NULL means never reviewed, which sorts first.
    --
    -- A counter rather than a timestamp on purpose: it cannot tie, so the rotation does not depend
    -- on the clock advancing between two reviews. Deliberately separate from updated_at, which the
    -- trigger below bumps on every edit.
    last_reviewed_seq INTEGER,
    created_at     TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at     TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (chat_id, word)
);

CREATE TRIGGER update_word_translations_updated_at
    BEFORE UPDATE ON word_translations
    BEGIN
        UPDATE word_translations SET updated_at = CURRENT_TIMESTAMP
        WHERE chat_id = NEW.chat_id AND word = NEW.word;
    END;

CREATE INDEX idx_word_translations_chat_id
    ON word_translations (chat_id);

-- Serves the review picker: learned words for a chat, least recently reviewed first.
CREATE INDEX idx_word_translations_review
    ON word_translations (chat_id, guessed_streak, last_reviewed_seq);

CREATE TABLE learning_batches
(
    chat_id INTEGER NOT NULL,
    word    TEXT    NOT NULL,

    PRIMARY KEY (chat_id, word),
    FOREIGN KEY (chat_id, word)
    REFERENCES word_translations (chat_id, word)
    ON DELETE CASCADE
    ON UPDATE CASCADE
);

CREATE INDEX idx_learning_batches_chat_id
    ON learning_batches (chat_id);

CREATE TABLE callback_data
(
    chat_id    INTEGER NOT NULL,
    uuid       TEXT    NOT NULL,
    data       TEXT    NOT NULL,
    expires_at TIMESTAMP NOT NULL,

    PRIMARY KEY (chat_id, uuid)
);

CREATE TABLE auth_confirmations
(
    chat_id    INTEGER NOT NULL,
    token      TEXT    NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    confirmed  INTEGER NOT NULL DEFAULT 0,

    PRIMARY KEY (chat_id, token)
);

CREATE TABLE statistics (
    chat_id INTEGER NOT NULL,
    date TEXT NOT NULL,
    words_guessed INTEGER NOT NULL DEFAULT 0,
    words_missed INTEGER NOT NULL DEFAULT 0,
    total_words_learned INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (chat_id, date)
);

CREATE INDEX idx_statistics_chat_id_date
    ON statistics (chat_id, date);
