# RT-Mail - AI Coding Agent Instructions

## Project Context

RT-Mail bridges managed email providers (Mailgun, SparkPost, SendGrid, Amazon SES) with Request Tracker (RT). The service receives incoming emails via webhooks and forwards them to RT's mail gateway API.

**Stack**: Go 1.25, net/http, Docker deployment

**Flow**: Email provider → webhook POST → handler extracts content → RT client maps to queue → posts to RT REST API

## CRITICAL Security Issue

**Missing Webhook Authentication** - Fix before production use

The service accepts webhook requests without authentication. Anyone who knows the webhook URLs can inject arbitrary emails into RT queues.

**Locations**: `mailgun/mailgun.go`, `sparkpost/sparkpost.go`, `sendgrid/sendgrid.go`, `ses/ses.go`

**Required**:
- Mailgun: Verify HMAC-SHA256 signatures
- SparkPost: Validate authentication headers
- SendGrid: Verify event webhook signatures
- SES: Verify SNS message signatures
- Add configuration fields for webhook secrets in `rt/rt.go`

## Pre-Commit Requirements

Before any `git commit`:

1. Run `gofumpt -w` on all modified .go files
2. Run `go test ./...` - all tests must pass
3. Remove trailing whitespace from edited lines
4. After running code generators, run `git status` and stage ALL generated files

## Git Usage

- Never use `git add -A`, `git add -a`, or `git add .`
- Add files explicitly by path
- Clean trailing whitespace before commits

## Testing

- Test actual code, not test code itself
- Don't duplicate production code into tests
- Don't test dependencies

## Example Data

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
├── ses/
│   └── ses.go          # Amazon SES webhook handler
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

## Technical Debt

Key items to address:
- Webhook authentication (critical)
- Structured logging improvements
- Test coverage for SparkPost
- Retry logic for RT requests
