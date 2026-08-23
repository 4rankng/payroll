-- Payrates represent a date-only timeline per project. These constraints make
-- corrupted intervals and duplicate live effective dates impossible even if an
-- application-level validation is bypassed or concurrent requests race.
ALTER TABLE payrates
    ADD CONSTRAINT chk_payrates_date_order CHECK (to_date IS NULL OR to_date >= from_date),
    ADD COLUMN live_from_date DATE GENERATED ALWAYS AS (
        CASE WHEN deleted_at IS NULL THEN from_date ELSE NULL END
    ) STORED,
    ADD COLUMN open_project_id BIGINT UNSIGNED GENERATED ALWAYS AS (
        CASE WHEN deleted_at IS NULL AND to_date IS NULL THEN project_id ELSE NULL END
    ) STORED,
    ADD UNIQUE KEY uq_payrates_live_project_from_date (project_id, live_from_date),
    ADD UNIQUE KEY uq_payrates_one_open_per_project (open_project_id);
