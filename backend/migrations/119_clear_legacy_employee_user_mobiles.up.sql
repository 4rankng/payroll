-- Clear legacy phone numbers copied onto employee rows in `users`.
--
-- Why
-- ---
-- A phone number belongs to the *employee* record (employees.mobile); the
-- account only borrows it through employees.user_id. A handful of `users` rows
-- created before that rule was enforced still carry a copy in users.mobile.
--
-- identity.Resolver already refuses to resolve such a row for an employee
-- (storesMobileInUserRow returns false for role='employee'), so these copies
-- are inert today. They are cleared because they are a trap: any future lookup
-- that forgets that guard would silently authenticate against the wrong number.
--
-- Only employee-role rows are touched. Admin / partner / adv_partner accounts
-- authoritatively store their number in users.mobile and must keep it.

-- Pre-check: the rows this migration clears (3 rows in the dev database).
-- Re-run after applying to confirm the list is empty.
-- SELECT id, username, mobile FROM users
--  WHERE role = 'employee' AND mobile IS NOT NULL AND mobile <> '';

UPDATE users
SET mobile = NULL
WHERE role = 'employee'
  AND mobile IS NOT NULL
  AND mobile <> '';