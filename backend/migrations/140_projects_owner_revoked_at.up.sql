-- PAYR-30: admin revokes the project creator's intrinsic owner access.
-- When set, created_by alone no longer grants project access — the user needs
-- an explicit project_users grant like any other manager. Grant/revoke rows
-- are untouched.
ALTER TABLE `projects`
  ADD COLUMN `owner_revoked_at` datetime(3) DEFAULT NULL;
