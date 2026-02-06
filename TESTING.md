# Testing Guide

## Running Tests

```bash
# Unit tests
go test ./cmd/... -v

# With race detection
go test ./cmd/... -v -race

# With coverage
go test ./cmd/... -v -coverprofile=coverage.out
go tool cover -html=coverage.out

# Integration tests (requires credentials)
export HY_TEST_API_KEY=sk_live_xxx
go test ./integration/... -v -tags=integration
```

## Test Patterns

### 1. Always Reset Global State

Viper config is global. Reset it in every test setup:

```go
func SetupTest(t *testing.T) *TestConfig {
    viper.Reset()  // IMPORTANT: prevents test contamination
    // ... rest of setup
}
```

### 2. Capture Stdout Correctly

Commands use `fmt.Println`, not Cobra's output. Capture real stdout:

```go
// ❌ Wrong - only captures Cobra's output
buf := new(bytes.Buffer)
cmd.SetOut(buf)

// ✅ Right - captures all stdout
oldStdout := os.Stdout
r, w, _ := os.Pipe()
os.Stdout = w

cmd.Execute()

w.Close()
os.Stdout = oldStdout
out, _ := io.ReadAll(r)
```

### 3. Mock Server Auth Patterns

Route handlers before checking auth (some endpoints like signed URLs skip auth):

```go
ts.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    // Check handlers FIRST
    key := r.Method + " " + r.URL.Path
    if handler, ok := ts.Handlers[key]; ok {
        handler(w, r)
        return
    }
    
    // THEN check auth for unhandled routes
    if r.Header.Get("Authorization") == "" {
        w.WriteHeader(http.StatusUnauthorized)
        return
    }
}))
```

### 4. Test Multi-Step Commands

If a command makes multiple API calls (e.g., GET then POST), mock all of them:

```go
// Build command does GET first to validate spec, then POST to trigger
tc.Server.HandleJSON("GET", "/productions/prod_xxx", http.StatusOK, production)
tc.Server.HandleJSON("POST", "/productions/prod_xxx/build", http.StatusAccepted, result)
```

### 5. Flag Registration Checklist

When adding a command flag:
- [ ] Add flag in command definition
- [ ] Register flag in `init()` function  
- [ ] Add test for flag behavior
- [ ] Update help text if needed

## Test File Structure

```
cmd/
├── testutil_test.go     # Shared test infrastructure
├── productions_test.go  # Production command tests
├── assets_test.go       # Asset command tests
├── keys_test.go         # Key command tests
└── thread_test.go       # Thread command tests

integration/
├── README.md            # Integration test setup
└── integration_test.go  # Live API tests
```

## Common Gotchas

| Issue | Solution |
|-------|----------|
| Tests see root help | Capture os.Stdout, not cmd buffer |
| Config leaks between tests | Call viper.Reset() in setup |
| Upload auth fails | Let mock route before auth check |
| Command behavior changed | Update mocks for all API calls |
| Missing flag error | Register in init(), add test |
