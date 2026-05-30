-- Migration: 035_notifications_sao_ke_fields.up.sql
ALTER TABLE notifications
    ADD COLUMN metadata TEXT NULL;