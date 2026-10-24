Reference: 1a2b3c4

# Acceptance tests: User authentication

| Criterion | Test | Command | Red proof |
|-----------|------|---------|-----------|
| AC-1 | TestLogin | go test ./internal/auth -run TestLogin | FAIL |

## Not covered

- AC-9: visual contrast, needs a human.

## Spec ambiguities

- Should a locked account return 401 or 423?
- [x] Is the session lifetime 24h? Resolved: yes.
- ~~Does logout invalidate all sessions?~~

### Conflicts with other features

None.
