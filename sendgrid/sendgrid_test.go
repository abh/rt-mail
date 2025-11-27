package sendgrid

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"go.askask.com/rt-mail/rt"
	"go.askask.com/rt-mail/testutil"
)

func TestSendgridReceiveHandler_Success(t *testing.T) {
	callCount := 0
	mockClient := &testutil.MockRTClient{
		PostmailFunc: func(recipient string, message string) error {
			callCount++
			return nil
		},
	}

	sg := &Sendgrid{RT: mockClient}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("envelope", `{"to":["test@example.com"],"from":"sender@example.com"}`)
	_ = writer.WriteField("email", "From: sender@example.com\nSubject: Test\n\nTest message")
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/sendgrid/mx", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	sg.ReceiveHandler(rr, req)

	testutil.AssertStatusCode(t, rr.Code, http.StatusNoContent)

	if callCount != 1 {
		t.Errorf("Expected Postmail called once, got %d calls", callCount)
	}
}

func TestSendgridReceiveHandler_MultipleRecipients(t *testing.T) {
	recipients := []string{}
	mockClient := &testutil.MockRTClient{
		PostmailFunc: func(recipient string, message string) error {
			recipients = append(recipients, recipient)
			return nil
		},
	}

	sg := &Sendgrid{RT: mockClient}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("envelope", `{"to":["test1@example.com","test2@example.com"],"from":"sender@example.com"}`)
	_ = writer.WriteField("email", "Test message")
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/sendgrid/mx", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	sg.ReceiveHandler(rr, req)

	testutil.AssertStatusCode(t, rr.Code, http.StatusNoContent)

	if len(recipients) != 2 {
		t.Errorf("Expected 2 recipients, got %d", len(recipients))
	}
}

func TestSendgridReceiveHandler_Integration(t *testing.T) {
	rtServer := testutil.NewMockRTServer(t)
	defer rtServer.Close()

	rtConfig := fmt.Sprintf(`{"rt-url": "%s", "queues": {"test@example.com": "test-queue"}}`, rtServer.URL)
	tmpfile := filepath.Join(t.TempDir(), "rt-mail-test.json")
	testutil.AssertNoError(t, os.WriteFile(tmpfile, []byte(rtConfig), 0o600))

	rtClient, err := rt.New(tmpfile)
	testutil.AssertNoError(t, err)

	sg := &Sendgrid{RT: rtClient}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("envelope", `{"to":["test@example.com"],"from":"sender@example.com"}`)
	_ = writer.WriteField("email", "Test message")
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/sendgrid/mx", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	sg.ReceiveHandler(rr, req)

	testutil.AssertStatusCode(t, rr.Code, http.StatusNoContent)
}
