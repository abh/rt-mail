# Managed email provider to Request Tracker Gateway

This tool implements the Sparkpost API for incoming mail and the RT API to
make it easy to use a managed email provider to receive and emails
with [Request Tracker](https://www.bestpractical.com/rt/).

## Configuration

See `rt-mail.json.sample` for an example configuration file.

The tool assumes that your "comment" address configured in RT is the same as
the correspondence address with "-comment" suffixed to the local part.

### Environment Variables

#### Amazon SES (Optional)

To enable Amazon SES support, configure these environment variables:

**RT-Mail Configuration:**
- `RT_SES_SNS_TOPIC_ARN` (required for SES) - The ARN of the SNS topic that receives SES notifications
  - Example: `arn:aws:sns:us-east-1:123456789012:ses-incoming-email`
  - If not set, SES handler is disabled

**AWS SDK Configuration:**

The AWS SDK requires credentials and region configuration. Use one of these methods:

1. **Environment variables:**
   - `AWS_REGION` or `AWS_DEFAULT_REGION` - AWS region (required)
     - Example: `us-east-1`
     - Must match the region where your S3 bucket is located
   - `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` - AWS credentials
   - `AWS_SESSION_TOKEN` - For temporary credentials (optional)

2. **AWS Profile:**
   - `AWS_PROFILE` - Name of profile from `~/.aws/credentials`
     - Example: `default` or `production`

3. **IAM Roles (recommended for AWS deployments):**
   - For EC2, ECS, EKS, or Lambda - attach an IAM role with S3 read permissions
   - No environment variables needed

**Required IAM Permissions:**
```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": ["s3:GetObject"],
    "Resource": "arn:aws:s3:::your-ses-bucket/*"
  }]
}
```

#### Other Providers

Mailgun, SparkPost, and SendGrid do not require environment variables. Configure webhook URLs in your provider's dashboard to point to the appropriate endpoints.

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

### Amazon SES

Configure SES to:
1. Store incoming emails in an S3 bucket using a receipt rule action
2. Publish notifications to an SNS topic
3. Subscribe the SNS topic to this webhook endpoint:

    /ses

The SES handler verifies SNS message signatures for security and automatically confirms SNS subscriptions. Emails are fetched from S3 (up to 50MB) and posted to RT for each recipient.

**Required**: Set the `RT_SES_SNS_TOPIC_ARN` environment variable (see Environment Variables section below).

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
