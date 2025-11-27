package ses

import (
	"encoding/json"
	"testing"

	"go.askask.com/rt-mail/testutil"
)

// TestBuildCanonicalString tests SNS signature canonical string building
func TestBuildCanonicalString(t *testing.T) {
	ses := &SES{}

	tests := []struct {
		name     string
		msg      *SNSMessage
		expected string
	}{
		{
			name: "Notification with subject",
			msg: &SNSMessage{
				Type:      "Notification",
				Message:   "test message",
				MessageID: "msg-123",
				Subject:   "Test Subject",
				Timestamp: "2025-01-01T00:00:00.000Z",
				TopicArn:  "arn:aws:sns:us-east-1:123:topic",
			},
			expected: "Message\ntest message\nMessageId\nmsg-123\nSubject\nTest Subject\nTimestamp\n2025-01-01T00:00:00.000Z\nTopicArn\narn:aws:sns:us-east-1:123:topic\nType\nNotification\n",
		},
		{
			name: "Notification without subject",
			msg: &SNSMessage{
				Type:      "Notification",
				Message:   "test message",
				MessageID: "msg-123",
				Timestamp: "2025-01-01T00:00:00.000Z",
				TopicArn:  "arn:aws:sns:us-east-1:123:topic",
			},
			expected: "Message\ntest message\nMessageId\nmsg-123\nTimestamp\n2025-01-01T00:00:00.000Z\nTopicArn\narn:aws:sns:us-east-1:123:topic\nType\nNotification\n",
		},
		{
			name: "SubscriptionConfirmation",
			msg: &SNSMessage{
				Type:         "SubscriptionConfirmation",
				Message:      "confirm",
				MessageID:    "msg-456",
				SubscribeURL: "https://sns.amazonaws.com/subscribe",
				Token:        "token-abc",
				Timestamp:    "2025-01-01T00:00:00.000Z",
				TopicArn:     "arn:aws:sns:us-east-1:123:topic",
			},
			expected: "Message\nconfirm\nMessageId\nmsg-456\nSubscribeURL\nhttps://sns.amazonaws.com/subscribe\nTimestamp\n2025-01-01T00:00:00.000Z\nToken\ntoken-abc\nTopicArn\narn:aws:sns:us-east-1:123:topic\nType\nSubscriptionConfirmation\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ses.buildCanonicalString(tt.msg)
			if got != tt.expected {
				t.Errorf("buildCanonicalString() mismatch\nGot:\n%q\nExpected:\n%q", got, tt.expected)
			}
		})
	}
}

// TestSNSHostPattern tests SNS host validation regex
func TestSNSHostPattern(t *testing.T) {
	tests := []struct {
		host  string
		valid bool
	}{
		{"sns.us-east-1.amazonaws.com", true},
		{"sns.eu-west-1.amazonaws.com", true},
		{"sns.ap-southeast-2.amazonaws.com", true},
		{"evil.com", false},
		{"sns.evil.com", false},
		{"amazonaws.com", false},
		{"sns..amazonaws.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			got := snsHostPattern.MatchString(tt.host)
			if got != tt.valid {
				t.Errorf("snsHostPattern.MatchString(%q) = %v, want %v", tt.host, got, tt.valid)
			}
		})
	}
}

// TestSESNotificationParsing tests parsing of SES notification JSON
func TestSESNotificationParsing(t *testing.T) {
	notifJSON := `{
		"notificationType": "Received",
		"receipt": {
			"action": {
				"type": "S3",
				"bucketName": "my-bucket",
				"objectKey": "emails/abc123"
			},
			"recipients": ["test@example.com", "test2@example.com"]
		},
		"mail": {
			"messageId": "msg-123",
			"source": "sender@example.com",
			"destination": ["test@example.com"]
		}
	}`

	var sesNotif SESNotification
	err := json.Unmarshal([]byte(notifJSON), &sesNotif)
	testutil.AssertNoError(t, err)

	if sesNotif.NotificationType != "Received" {
		t.Errorf("Expected NotificationType=Received, got %s", sesNotif.NotificationType)
	}
	if sesNotif.Receipt.Action.Type != "S3" {
		t.Errorf("Expected Action.Type=S3, got %s", sesNotif.Receipt.Action.Type)
	}
	if sesNotif.Receipt.Action.BucketName != "my-bucket" {
		t.Errorf("Expected BucketName=my-bucket, got %s", sesNotif.Receipt.Action.BucketName)
	}
	if len(sesNotif.Receipt.Recipients) != 2 {
		t.Errorf("Expected 2 recipients, got %d", len(sesNotif.Receipt.Recipients))
	}
}

// TestMaxEmailSizeConstant tests that the size limit is reasonable
func TestMaxEmailSizeConstant(t *testing.T) {
	expected := 50 * 1024 * 1024 // 50MB
	if maxEmailSize != expected {
		t.Errorf("maxEmailSize = %d, want %d", maxEmailSize, expected)
	}
}

// Note: Full integration tests with SNS signature verification and S3 mocking
// would require:
// 1. Mocking SNS certificate fetching and validation
// 2. Implementing S3 client interface for mocking
// 3. Creating test fixtures for AWS SNS messages
//
// For now, the critical functions (canonical string building, host validation,
// and JSON parsing) are tested. Full end-to-end testing would be done in
// staging environment with real AWS services.
