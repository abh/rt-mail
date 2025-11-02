# RT-Mail - Project Overview

## What This Project Does

RT-Mail is a gateway service that bridges managed email service providers (Mailgun, SparkPost, SendGrid) with Request Tracker (RT), an open-source issue tracking system. It receives incoming emails via webhooks from these providers and forwards them to RT's mail gateway API, enabling organizations to use modern email infrastructure while maintaining their RT-based ticketing workflow.

### Key Components

- **Main Service** (`main.go`): HTTP server with REST API endpoints for each email provider
- **RT Client** (`rt/rt.go`): Handles communication with Request Tracker's REST API
- **Provider Handlers**: Separate packages for Mailgun, SparkPost, and SendGrid webhook processing
- **Configuration**: JSON-based mapping of email addresses to RT queues

### Architecture Flow

1. Email provider receives incoming email
2. Provider sends webhook POST request to rt-mail endpoint (`/mg/mx/mime`, `/spark/mx`, or `/sendgrid/mx`)
3. Handler extracts recipient and email content from provider-specific format
4. RT client maps recipient to queue and action (correspond/comment)
5. Email is posted to RT's mail gateway endpoint
6. Response status returned to email provider

## Tech Stack

- **Language**: Go 1.21.3
- **Web Framework**: go-json-rest (REST API middleware)
- **Dependencies**:
  - `github.com/SparkPost/gosparkpost` - SparkPost event handling
  - `github.com/ant0ine/go-json-rest` - REST framework
- **Deployment**: Docker with multi-stage build
- **CI/CD**: GitHub Actions (Docker image publishing to GHCR)

## Project Structure

```
rt-mail/
├── main.go              # Application entry point and routing
├── rt/
│   ├── rt.go           # RT client and queue mapping logic
│   └── rt_test.go      # Address-to-queue mapping tests
├── mailgun/
│   └── mailgun.go      # Mailgun webhook handler
├── sparkpost/
│   └── sparkpost.go    # SparkPost relay/event handler
├── sendgrid/
│   └── sendgrid.go     # SendGrid webhook handler
├── rt-mail.json.sample # Configuration template
└── Dockerfile          # Multi-stage Docker build
```

## Configuration

Configuration via `rt-mail.json`:

```json
{
  "rt-url": "https://rt.example.com/REST/1.0/NoAuth/mail-gateway",
  "queues": {
    "sales": "sales-queue",
    "sales@widgets.example.com": "sales-widgets",
    "help-comment": "help"
  }
}
```

- Maps email addresses (or local parts) to RT queue names
- Supports `-comment` suffix convention for comment vs. correspondence distinction
- Case-insensitive matching with fallback to local part only

## Current State Assessment

### Strengths

1. **Clear separation of concerns**: Each provider has its own package
2. **Provider abstraction**: Common interface pattern for extensibility
3. **Flexible queue mapping**: Supports both full addresses and local parts
4. **Production-ready deployment**: Docker containerization with non-root user
5. **HTTP middleware stack**: Logging, recovery, gzip compression included

### Technical Debt

1. **Deprecated dependencies**: `go-json-rest` is unmaintained (last commit 2017)
2. **Deprecated Go APIs**: Uses `ioutil.ReadFile` and `ioutil.ReadAll` (deprecated since Go 1.16)
3. **Inconsistent error handling**: Mix of logging and returning errors
4. **Limited observability**: No structured logging or metrics
5. **No input validation**: Missing size limits, content validation
6. **Hardcoded magic numbers**: Size limits (50MB) scattered throughout code
7. **No retry logic**: RT requests fail immediately without retries
8. **Weak RT API parsing**: Relies on string matching for "failure" detection

---

# Top 5 Recommended Improvements

## 1. Migrate from go-json-rest to Standard Library or Modern Framework

**Priority**: HIGH
**Impact**: Security, Maintainability, Performance

### Problem

The `go-json-rest` library is unmaintained (last updated 2017) and doesn't support modern Go features. This creates security risks and prevents leveraging improvements in Go's standard library.

### Solution

Migrate to Go's native `net/http` with modern routing (e.g., `gorilla/mux` or `chi`), or use a maintained framework like `fiber` or `echo`.

**Recommended approach**: Use standard library `http.ServeMux` (enhanced in Go 1.22+) or `chi` for minimal changes.

### Benefits

- Active security patches and Go version compatibility
- Better performance with modern HTTP/2 and HTTP/3 support
- Reduced dependency attack surface
- Access to modern middleware ecosystem
- Easier debugging and community support

### Implementation Estimate

- **Effort**: 4-6 hours
- **Risk**: Medium (requires testing all provider endpoints)
- **Files affected**: `main.go`, all provider packages

---

## 2. Replace Deprecated `ioutil` Functions with Modern Equivalents

**Priority**: MEDIUM
**Impact**: Code Quality, Future Compatibility

### Problem

The code uses deprecated functions from the `ioutil` package:
- `ioutil.ReadFile` (rt/rt.go:52)
- `ioutil.ReadAll` (rt/rt.go:138, sparkpost/sparkpost.go:37)

These were deprecated in Go 1.16 and may be removed in future versions.

### Solution

Replace with standard library equivalents:
- `ioutil.ReadFile` → `os.ReadFile`
- `ioutil.ReadAll` → `io.ReadAll`

### Benefits

- Future-proof against Go version deprecations
- Signals code is actively maintained
- No performance overhead (direct replacements)
- Improves IDE warnings/linting

### Implementation Estimate

- **Effort**: 15-30 minutes
- **Risk**: Very Low (drop-in replacements)
- **Files affected**: `rt/rt.go`, `sparkpost/sparkpost.go`

---

## 3. Implement Structured Logging and Observability

**Priority**: HIGH
**Impact**: Production Operations, Debugging

### Problem

Current logging uses inconsistent approaches:
- Mix of `log.Printf`, `fmt.Printf`, and direct writing to stdout
- No log levels (debug/info/warn/error)
- No structured fields for filtering/searching
- No correlation IDs for request tracing
- No metrics for monitoring

Examples:
```go
// sparkpost/sparkpost.go:30
fmt.Printf("POST to '%s': %#v\n\n", r.URL.String(), r)

// rt/rt.go:126
log.Printf("posting to queue '%s' (action: '%s')", queue, action)
```

### Solution

Implement structured logging with a library like `slog` (Go 1.21+), `zap`, or `zerolog`:

```go
logger.Info("posting to RT queue",
    "queue", queue,
    "action", action,
    "recipient", recipient,
    "request_id", requestID)
```

Add observability features:
- Request ID propagation (middleware)
- Log levels configurable via environment
- Structured error context
- Optional metrics export (Prometheus)

### Benefits

- Searchable, filterable logs in production
- Easy correlation of requests across components
- Better debugging of production issues
- Enables alerting and monitoring
- Clean separation of debug vs. production logs

### Implementation Estimate

- **Effort**: 6-8 hours
- **Risk**: Low (additive change)
- **Files affected**: All Go files

---

## 4. Add Robust Error Handling and Retry Logic

**Priority**: HIGH
**Impact**: Reliability, Data Loss Prevention

### Problem

Current implementation has several reliability issues:

1. **No retry logic**: RT API failures immediately fail the request
2. **Weak error detection**: Checks for substring "failure" in RT response
3. **No circuit breaking**: Cascading failures if RT is down
4. **Silent failures**: Some errors logged but not propagated (sendgrid/sendgrid.go:66)
5. **No timeout configuration**: Hardcoded 10-second timeout

### Solution

Implement comprehensive error handling:

```go
// Exponential backoff retry for transient failures
func (rt *RT) PostmailWithRetry(ctx context.Context, recipient, message string) error {
    retryConfig := backoff.NewExponentialBackOff()
    retryConfig.MaxElapsedTime = 30 * time.Second

    return backoff.Retry(func() error {
        err := rt.Postmail(recipient, message)
        if err != nil && isTransient(err) {
            logger.Warn("RT request failed, retrying", "error", err)
            return err
        }
        if err != nil {
            return backoff.Permanent(err) // Don't retry
        }
        return nil
    }, retryConfig)
}
```

Add proper RT response parsing:
- Parse RT's actual response format (not substring matching)
- Distinguish between client errors (4xx) and server errors (5xx)
- Return structured errors with context

### Benefits

- Resilient to transient network failures
- Prevents email loss during RT downtime
- Better error messages for operators
- Graceful degradation
- Proper HTTP status codes to email providers

### Implementation Estimate

- **Effort**: 8-10 hours
- **Risk**: Medium (requires careful testing)
- **Files affected**: `rt/rt.go`, all provider handlers

---

## 5. Add Comprehensive Testing and CI Quality Gates

**Priority**: MEDIUM
**Impact**: Code Quality, Regression Prevention

### Problem

Current test coverage is minimal:
- Only 2 test files (`rt/rt_test.go`, `main_test.go`)
- `main_test.go` has a single basic test
- No integration tests
- No test coverage measurement
- No linting in CI
- No provider handler tests

### Solution

Implement comprehensive testing strategy:

**Unit Tests**:
- Test all provider handlers with mock RT client
- Test RT client with mock HTTP server
- Test queue mapping edge cases
- Test error conditions

**Integration Tests**:
- Docker Compose with mock RT server
- End-to-end tests for each provider webhook format
- Test retry logic and failure scenarios

**CI Enhancements**:
```yaml
# Add to .github/workflows/
- name: Run tests
  run: go test -v -race -coverprofile=coverage.out ./...

- name: Check coverage
  run: |
    go tool cover -func=coverage.out
    # Fail if coverage < 70%

- name: Run linters
  uses: golangci/golangci-lint-action@v3
  with:
    version: latest
```

**Static Analysis**:
- Enable `golangci-lint` with standard linters
- Add `gosec` for security scanning
- Add `go vet` checks

### Benefits

- Catch regressions before deployment
- Document expected behavior
- Enable confident refactoring
- Reduce production bugs
- Faster code review process
- Security vulnerability detection

### Implementation Estimate

- **Effort**: 12-16 hours
- **Risk**: Low (additive)
- **Files affected**: New test files, `.github/workflows/`

---

## Additional Recommendations (Lower Priority)

### 6. Add Configuration Validation on Startup

Validate `rt-mail.json` format and RT connectivity at startup to fail fast.

### 7. Implement Health Check Improvements

Current `/healthz` returns 204 without checking RT connectivity. Add deep health checks.

### 8. Add Request Size Limits Configuration

Make the 50MB limit configurable via environment variables or config file.

### 9. Security Hardening

- Add webhook signature validation (Mailgun/SendGrid support this)
- Implement rate limiting per provider
- Add HTTPS requirement in production

### 10. Documentation

- API documentation with OpenAPI/Swagger spec
- Deployment guide with Kubernetes manifests
- Troubleshooting runbook

---

## Getting Started for Developers

### Prerequisites

- Go 1.21.3+
- Docker (optional, for containerized deployment)
- Access to RT instance with REST API enabled

### Local Development

```bash
# Clone and build
git clone <repo-url>
cd rt-mail
go mod download
go build

# Copy and configure
cp rt-mail.json.sample rt-mail.json
# Edit rt-mail.json with your RT URL and queue mappings

# Run locally
./rt-mail -config=rt-mail.json -listen=:8081

# Test with curl
curl -X POST http://localhost:8081/healthz
```

### Testing

```bash
# Run tests
go test ./...

# Run specific test
go test -v ./rt -run TestAddressQueueMap

# With coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Docker Deployment

```bash
docker build -t rt-mail .
docker run -p 8081:8002 \
  -v $(pwd)/rt-mail.json:/etc/rt-mail/config.json \
  rt-mail
```

---

## License

MIT License - See LICENSE file for details.

## Contributing

This codebase would benefit from the improvements outlined above. When contributing:

1. Follow Go conventions and `gofmt` formatting
2. Add tests for new functionality
3. Update this documentation for architectural changes
4. Keep provider handlers independent and testable

## Support

For issues with:
- **Request Tracker**: See [RT Documentation](https://docs.bestpractical.com/rt/)
- **Email Providers**: Check respective API documentation (Mailgun, SparkPost, SendGrid)
- **This Gateway**: Open GitHub issue with logs and configuration (redact credentials)
