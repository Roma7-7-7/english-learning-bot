CREATE TABLE word_translations
(
    chat_id        INT          NOT NULL,
    word           VARCHAR(255) NOT NULL,
    translation    VARCHAR(255) NOT NULL,
    description    varchar(1024),
    guessed_streak INT          NOT NULL DEFAULT 0,
    to_review      BOOL NOT NULL DEFAULT FALSE,
    created_at     TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at     TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (chat_id, word)
);

CREATE
    OR REPLACE FUNCTION update_updated_at_column()
    RETURNS TRIGGER AS
$$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;

$$ language 'plpgsql';

CREATE TRIGGER update_word_translations_updated_at
    BEFORE UPDATE
    ON word_translations
    FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

CREATE INDEX idx_word_translations_chat_id
    ON word_translations (chat_id);

CREATE TABLE learning_batches
(
    chat_id INT          NOT NULL,
    word    VARCHAR(255) NOT NULL,

    PRIMARY KEY (chat_id, word)
);

ALTER TABLE learning_batches
    ADD FOREIGN KEY (chat_id, word)
    REFERENCES word_translations (chat_id, word)
    ON DELETE CASCADE
    ON UPDATE CASCADE;

CREATE INDEX idx_learning_batches_chat_id
    ON learning_batches (chat_id);

CREATE TABLE callback_data
(
    chat_id    INT          NOT NULL,
    uuid       UUID         NOT NULL,
    data       JSONB        NOT NULL,
    expires_at TIMESTAMP    NOT NULL,

    PRIMARY KEY (chat_id, uuid)
);

CREATE TABLE auth_confirmations
(
    chat_id    INT          NOT NULL,
    token      UUID         NOT NULL,
    expires_at TIMESTAMP    NOT NULL,
    confirmed  BOOL         NOT NULL DEFAULT FALSE,

    PRIMARY KEY (chat_id, token)
);

CREATE TABLE daily_word_statistics (
    chat_id INT NOT NULL,
    date DATE NOT NULL,
    words_guessed INT NOT NULL DEFAULT 0,
    words_missed INT NOT NULL DEFAULT 0,
    words_to_review INT NOT NULL DEFAULT 0,
    total_words_guessed INT NOT NULL DEFAULT 0,
    avg_guesses_to_success FLOAT NOT NULL DEFAULT 0,
    longest_streak INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (chat_id, date)
);

CREATE INDEX idx_daily_word_statistics_chat_id_date 
    ON daily_word_statistics (chat_id, date);
