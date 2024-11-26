CREATE TABLE word_translations
(
    chat_id        INT          NOT NULL,
    word           VARCHAR(255) NOT NULL,
    translation    VARCHAR(255) NOT NULL,
    description    varchar(1024),
    guessed_streak INT          NOT NULL DEFAULT 0,
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

CREATE INDEX idx_learning_batches_chat_id
    ON learning_batches (chat_id);