ALTER TABLE payrates
    DROP INDEX uq_payrates_one_open_per_project,
    DROP INDEX uq_payrates_live_project_from_date,
    DROP CHECK chk_payrates_date_order,
    DROP COLUMN open_project_id,
    DROP COLUMN live_from_date;
