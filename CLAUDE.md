# RT-Mail - AI Coding Agent Instructions

## Project Context

RT-Mail bridges managed email providers (Mailgun, SparkPost, SendGrid) with Request Tracker (RT). The service receives incoming emails via webhooks and forwards them to RT's mail gateway API, enabling organizations to use modern email infrastructure with RT-based ticketing.

**Stack**: Go 1.21.3, go-json-rest (REST middleware), Docker deployment

**Flow**: Email provider → webhook POST → handler extracts content → RT client maps to queue → posts to RT REST API

## CRITICAL Security Issue

**Missing Webhook Authentication** - Fix before production use

The service accepts webhook requests without authentication. Anyone who knows the webhook URLs can inject arbitrary emails into RT queues.

**Locations**: `mailgun/mailgun.go:23`, `sparkpost/sparkpost.go:68,28`, `sendgrid/sendgrid.go:26`

**Required**:
- Mailgun: Verify HMAC-SHA256 signatures
- SparkPost: Validate authentication headers
- SendGrid: Verify event webhook signatures
- Add configuration fields for webhook secrets in `rt/rt.go`

**See TODO.md for detailed implementation steps**

## High-Priority Fixes

### Replace Deprecated io/ioutil

**Locations**: `rt/rt.go:6,52,137`, `sparkpost/sparkpost.go:6,37`

The code uses deprecated `io/ioutil` package (deprecated since Go 1.16):

```go
// Replace
ioutil.ReadFile()  → os.ReadFile()
ioutil.ReadAll()   → io.ReadAll()
```

**Effort**: 15-30 minutes

### Fix Unchecked Error (sendgrid/sendgrid.go:42)

The Sendgrid handler unmarshals envelope JSON without checking errors. If envelope is malformed, `result.To` will be empty and handler silently fails.

Add error checking and validate recipients exist.

**Effort**: 15 minutes

### Remove Debug fmt.Printf

**Locations**: All provider handlers

Replace `fmt.Printf` with structured logging (`log/slog`). Current debug prints expose sensitive data (request headers, email content) in production logs.

Add log levels (DEBUG, INFO, ERROR) configurable via environment.

### Add Retry Logic

RT requests fail immediately without retries. Implement exponential backoff for transient failures (network errors, 5xx responses). Don't retry client errors (4xx).

**See TODO.md for detailed implementation**

## Development Conventions

### Pre-Commit Requirements

Before any `git commit`:

1. Run `gofumpt -w` on all modified .go files
2. Run `go test ./...` - all tests must pass
3. Remove trailing whitespace from edited lines
4. After running code generators (sqlc, gRPC, enumer), run `git status` and stage ALL generated files

### Git Usage

- Never use `git add -A`, `git add -a`, or `git add .`
- Add files explicitly by path
- Clean trailing whitespace before commits

### Testing

- Test actual code, not test code itself
- Don't duplicate production code into tests
- Don't test dependencies

### Example Data

Use appropriate example domains and IPs:
- Email: `email@example.com` (not `some@email.com`)
- IP: RFC 5737 addresses (192.0.2.1, not 1.1.1.1)

## Project Structure

```
rt-mail/
├── main.go              # Entry point, routing
├── rt/
│   ├── rt.go           # RT client, queue mapping
│   └── rt_test.go      # Mapping tests
├── mailgun/
│   └── mailgun.go      # Mailgun webhook handler
├── sparkpost/
│   └── sparkpost.go    # SparkPost webhook handler
├── sendgrid/
│   └── sendgrid.go     # SendGrid webhook handler
├── rt-mail.json.sample # Configuration template
└── Dockerfile          # Multi-stage build
```

## Configuration

**File**: `rt-mail.json`

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

Maps email addresses (or local parts) to RT queue names:
- Supports `-comment` suffix for comment vs. correspondence
- Case-insensitive matching
- Falls back to local part if full address not found

## Architecture Strengths

1. Clean separation of concerns (providers in separate packages)
2. Provider abstraction via common interface
3. Flexible queue mapping (full addresses and local parts)
4. Docker containerization with non-root user
5. HTTP middleware stack (logging, recovery, gzip)

## Technical Debt and Improvements

See TODO.md for prioritized actionable items with effort estimates.

Key items:
- Webhook authentication (critical)
- Deprecated ioutil usage (high)
- Structured logging (medium)
- Test coverage (medium)
- Framework migration from unmaintained go-json-rest (evaluate)
