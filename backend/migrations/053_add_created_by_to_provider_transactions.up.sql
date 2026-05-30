-- +migrate Up
ALTER TABLE provider_transactions
    ADD COLUMN created_by BIGINT UNSIGNED NULL AFTER entity_id,
    ADD CONSTRAINT fk_provider_transactions_creator FOREIGN KEY (created_by) REFERENCES users (id);
