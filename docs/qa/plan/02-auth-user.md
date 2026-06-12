# 02 — Auth & User Management

**Priority**: P1 | **API Prefix**: `/api/v1/auth`, `/api/v1/users` | **Risk Level**: Medium

## Business Rules

- JWT-based authentication with token expiry
- Casbin RBAC with roles: `admin`, `partner`
- Password minimum length enforced
- Duplicate usernames rejected

## State Machine

```
User: [created] → active → [deleted]
                ↕ (update profile/password)
```

## Test Scenarios

### F01 — Auth & Login

| # | Scenario | Type | Steps | Expected Result |
|---|----------|------|-------|-----------------|
| F01-01 | Admin login with valid credentials | Happy | POST `/auth/login` with admin username/password | 200, JWT token returned, token contains admin role |
| F01-02 | Partner login with valid credentials | Happy | POST `/auth/login` with partner username/password | 200, JWT token returned, token contains partner role |
| F01-03 | Login with wrong password | Negative | POST `/auth/login` with valid username + wrong password | 401, "invalid credentials" |
| F01-04 | Login with non-existent username | Negative | POST `/auth/login` with unknown username | 401, "invalid credentials" |
| F01-05 | Access protected route without token | Negative | GET `/users` without Authorization header | 401 Unauthorized |
| F01-06 | Access admin route as partner | Negative | GET `/admin/*` with partner token | 403 Forbidden |
| F01-07 | Token expiry behavior | Edge | Wait for token expiry, then call protected endpoint | 401, must re-login |

### F02 — User CRUD

| # | Scenario | Type | Steps | Expected Result |
|---|----------|------|-------|-----------------|
| F02-01 | Create admin user | Happy | POST `/users` with role=admin, username, password | 201, user created with admin role |
| F02-02 | Create partner user | Happy | POST `/users` with role=partner | 201, user created with partner role |
| F02-03 | Get user profile | Happy | GET `/users/:id` | 200, returns user details (no password) |
| F02-04 | Update user profile | Happy | PUT `/users/:id` with new fullname | 200, fullname updated |
| F02-05 | Create duplicate username | Negative | POST `/users` with existing username | 400, "username already exists" |
| F02-06 | Create user with short password | Negative | POST `/users` with password < minimum | 400, validation error |
| F02-07 | Delete user | Happy | DELETE `/users/:id` | 200, user soft-deleted |
| F02-08 | Cross-module: User visible in project users | Integration | Create user → assign to project → list project users | User appears in project user list |

## Automated Test Reference

- Integration test: `backend/tests/integration/flow_auth_user.go`
