-- 086: per-user JWT invalidation timestamp. When non-NULL, AuthService.ValidateToken
-- rejects any access token whose IssuedAt is older than this value. This makes the
-- admin "revoke all sessions" action (AuthService.RevokeUserTokens) actually take
-- effect, instead of the prior behavior where it only wrote an audit row and left
-- every issued token valid for its full 14-day TTL.
--   NULL  = no global revocation in effect (normal state)
--   <ts>  = tokens issued before this instant are invalid; newer tokens are unaffected
ALTER TABLE users
    ADD COLUMN tokens_invalid_before DATETIME(3) NULL COMMENT 'when set, any JWT issued before this instant is rejected on validate; used by RevokeUserTokens' AFTER last_login;
