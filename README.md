# Managed email provider to Request Tracker Gateway

This tool implements the Sparkpost API for incoming mail and the RT API to
make it easy to use a managed email provider to receive and emails
with [Request Tracker](https://www.bestpractical.com/rt/).

## Configuration

See `rt-mail.json.sample` for an example configuration file.

The tool assumes that your "comment" address configured in RT is the same as
the correspondence address with "-comment" suffixed to the local part.

## Run

    ./rt-mail -listen=:8081 -config=rt-mail.json

## Email service provider configuration

There's a unique path for each email service provider API. For each of them
prefix the path with the host and port that rt-mail is running on.

### Mailgun

Configure Mailgun to `forward` mails to

    /mg/mx/mime

### SparkPost

Configure SparkPost to relay messages to

    /spark/mx

### Sendgrid

Configure Sendgrid to relay messages, you'll need to enable [full MIME emails](https://sendgrid.com/docs/for-developers/parsing-email/setting-up-the-inbound-parse-webhook/)

    /sendgrid/mx

## Development

### Quick Start

```bash
# Install development tools
make install-tools

# Run tests
make test

# Run linters
make lint

# Format code
make fmt

# Run all checks (fmt, vet, lint, test)
make check

# Build the binary
make build

# Run locally (requires rt-mail.json)
make run
```

### Makefile Targets

Run `make help` to see all available targets:

- `make build` - Build the rt-mail binary
- `make test` - Run all tests
- `make test-coverage` - Run tests with coverage report
- `make lint` - Run linters
- `make fmt` - Format code
- `make check` - Run all checks (recommended before committing)
- `make install-tools` - Install development tools (golangci-lint)
- `make clean` - Clean build artifacts

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# View HTML coverage report
make test-coverage-html

# Run tests for a specific package
go test -v ./rt
```

### Continuous Integration

The project uses GitHub Actions for CI/CD:

- **Test workflow** (`.github/workflows/test.yml`) - Runs tests, linting, and builds on every push/PR
- **Docker workflow** (`.github/workflows/docker-publish.yml`) - Builds and publishes Docker images

## TODO

- support more providers
- Capture bounce events
  - https://github.com/SparkPost/event-data/blob/master/sql/tables.ddl
