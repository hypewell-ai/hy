# Integration Tests

Integration tests run against the live Hypewell Studio API using a dedicated test workspace.

## Setup

### 1. Create Test Workspace

The test workspace `ws_integration_test` is used for all integration tests. Create it once:

```bash
# Using Firebase CLI
firebase firestore:set workspaces/ws_integration_test \
  --project hypewell-prod \
  --data '{"id":"ws_integration_test","name":"Integration Tests","slug":"integration-test","plan":"free"}'
```

Or manually in Firebase Console → Firestore → workspaces → Add Document with ID `ws_integration_test`.

### 2. Create Test API Key

```bash
# After logging in with a real account
hy keys create --name "Integration Tests" --scopes "productions:read,productions:write,assets:read,assets:write,thread:read,thread:write"
```

Save the key to `~/.config/hy/test-key` (gitignored).

### 3. Configure Test Environment

```bash
export HY_TEST_API_KEY=$(cat ~/.config/hy/test-key)
export HY_TEST_WORKSPACE_ID=ws_integration_test
```

## Running Tests

```bash
# Run all integration tests
go test ./integration/... -v

# Run specific test
go test ./integration/... -v -run TestProductionsIntegration
```

## Test Isolation

- All tests use `ws_integration_test` workspace
- Tests clean up after themselves (delete created resources)
- Use `--dry-run` and `--validate-only` flags where possible
- Never trigger actual Cloud Build runs (use validate-only)

## Flags for Safe Testing

| Command | Flag | Effect |
|---------|------|--------|
| `hy productions create` | `--dry-run` | Validates without creating |
| `hy productions build` | `--validate-only` | Checks spec without building |
| `hy assets upload` | `--dry-run` | Validates without uploading |

## Test Data Cleanup

Integration tests should clean up created resources. If cleanup fails, run:

```bash
# List test productions
hy productions list --workspace ws_integration_test

# Delete manually
hy productions delete <id> --force
```
