package mailgun

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

func TestMailgunReceiveHandler_Success(t *testing.T) {
	mockClient := &testutil.MockRTClient{
		PostmailFunc: func(recipient string, message string) error {
			if recipient == "" || message == "" {
				t.Error("Expected recipient and message to be set")
			}
			return nil
		},
	}

	mg := &Mailgun{RT: mockClient}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("recipient", "test@example.com")
	_ = writer.WriteField("body-mime", "From: sender@example.com\nSubject: Test\n\nTest message")
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/mg/mx/mime", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	mg.ReceiveHandler(rr, req)

	testutil.AssertStatusCode(t, rr.Code, http.StatusNoContent)
}

func TestMailgunReceiveHandler_NotFound(t *testing.T) {
	mockClient := &testutil.MockRTClient{
		PostmailFunc: func(recipient string, message string) error {
			return &rt.Error{NotFound: true}
		},
	}

	mg := &Mailgun{RT: mockClient}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("recipient", "unknown@example.com")
	_ = writer.WriteField("body-mime", "Test")
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/mg/mx/mime", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	mg.ReceiveHandler(rr, req)

	testutil.AssertStatusCode(t, rr.Code, http.StatusNotFound)
}

func TestMailgunReceiveHandler_Integration(t *testing.T) {
	rtServer := testutil.NewMockRTServer(t)
	defer rtServer.Close()

	rtConfig := fmt.Sprintf(`{"rt-url": "%s", "queues": {"test@example.com": "test-queue"}}`, rtServer.URL)
	tmpfile := filepath.Join(t.TempDir(), "rt-mail-test.json")
	testutil.AssertNoError(t, os.WriteFile(tmpfile, []byte(rtConfig), 0o600))

	rtClient, err := rt.New(tmpfile)
	testutil.AssertNoError(t, err)

	mg := &Mailgun{RT: rtClient}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("recipient", "test@example.com")
	_ = writer.WriteField("body-mime", "Test message")
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/mg/mx/mime", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	mg.ReceiveHandler(rr, req)

	testutil.AssertStatusCode(t, rr.Code, http.StatusNoContent)
}
