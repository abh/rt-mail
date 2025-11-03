package sendgrid

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/ant0ine/go-json-rest/rest/test"

	"go.askask.com/rt-mail/rt"
)

// mockRT implements a mock RT client for testing
type mockRT struct {
	postmailFunc func(recipient string, message string) error
}

func (m *mockRT) Postmail(recipient string, message string) error {
	if m.postmailFunc != nil {
		return m.postmailFunc(recipient, message)
	}
	return nil
}

func TestSendgridReceiveHandler_Success(t *testing.T) {
	callCount := 0
	mockClient := &mockRT{
		postmailFunc: func(recipient string, message string) error {
			callCount++
			if recipient == "" {
				t.Error("Expected recipient to be set")
			}
			if message == "" {
				t.Error("Expected message to be set")
			}
			return nil
		},
	}

	sg := &Sendgrid{RT: mockClient}

	api := rest.NewApi()
	router, err := rest.MakeRouter(
		rest.Post("/sendgrid/mx", sg.ReceiveHandler),
	)
	if err != nil {
		t.Fatal(err)
	}
	api.SetApp(router)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("envelope", `{"to":["test@example.com"],"from":"sender@example.com"}`)
	_ = writer.WriteField("email", "From: sender@example.com\nTo: test@example.com\nSubject: Test\n\nTest message")
	writer.Close()

	req, err := http.NewRequest("POST", "/sendgrid/mx", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	recorded := test.RunRequest(t, api.MakeHandler(), req)
	recorded.CodeIs(204)

	if callCount != 1 {
		t.Errorf("Expected Postmail to be called once, got %d calls", callCount)
	}
}

func TestSendgridReceiveHandler_MultipleRecipients(t *testing.T) {
	recipients := []string{}
	mockClient := &mockRT{
		postmailFunc: func(recipient string, message string) error {
			recipients = append(recipients, recipient)
			return nil
		},
	}

	sg := &Sendgrid{RT: mockClient}

	api := rest.NewApi()
	router, err := rest.MakeRouter(
		rest.Post("/sendgrid/mx", sg.ReceiveHandler),
	)
	if err != nil {
		t.Fatal(err)
	}
	api.SetApp(router)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("envelope", `{"to":["test1@example.com","test2@example.com","test3@example.com"],"from":"sender@example.com"}`)
	_ = writer.WriteField("email", "Test message")
	writer.Close()

	req, err := http.NewRequest("POST", "/sendgrid/mx", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	recorded := test.RunRequest(t, api.MakeHandler(), req)
	recorded.CodeIs(204)

	if len(recipients) != 3 {
		t.Errorf("Expected 3 recipients, got %d", len(recipients))
	}
}

func TestSendgridReceiveHandler_AllNotFound(t *testing.T) {
	mockClient := &mockRT{
		postmailFunc: func(recipient string, message string) error {
			return &rt.Error{
				NotFound: true,
			}
		},
	}

	sg := &Sendgrid{RT: mockClient}

	api := rest.NewApi()
	router, err := rest.MakeRouter(
		rest.Post("/sendgrid/mx", sg.ReceiveHandler),
	)
	if err != nil {
		t.Fatal(err)
	}
	api.SetApp(router)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("envelope", `{"to":["unknown@example.com"],"from":"sender@example.com"}`)
	_ = writer.WriteField("email", "Test message")
	writer.Close()

	req, err := http.NewRequest("POST", "/sendgrid/mx", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	recorded := test.RunRequest(t, api.MakeHandler(), req)
	recorded.CodeIs(404)
}

func TestSendgridReceiveHandler_PartialSuccess(t *testing.T) {
	// First recipient succeeds, second fails with NotFound, third succeeds
	callCount := 0
	mockClient := &mockRT{
		postmailFunc: func(recipient string, message string) error {
			callCount++
			if recipient == "unknown@example.com" {
				return &rt.Error{
					NotFound: true,
				}
			}
			return nil
		},
	}

	sg := &Sendgrid{RT: mockClient}

	api := rest.NewApi()
	router, err := rest.MakeRouter(
		rest.Post("/sendgrid/mx", sg.ReceiveHandler),
	)
	if err != nil {
		t.Fatal(err)
	}
	api.SetApp(router)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("envelope", `{"to":["test1@example.com","unknown@example.com","test3@example.com"],"from":"sender@example.com"}`)
	_ = writer.WriteField("email", "Test message")
	writer.Close()

	req, err := http.NewRequest("POST", "/sendgrid/mx", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	recorded := test.RunRequest(t, api.MakeHandler(), req)
	// Should return 204 because at least one recipient succeeded
	recorded.CodeIs(204)
}

func TestSendgridReceiveHandler_ServerError(t *testing.T) {
	callCount := 0
	mockClient := &mockRT{
		postmailFunc: func(recipient string, message string) error {
			callCount++
			if callCount == 1 {
				// First call succeeds
				return nil
			}
			// Second call fails with server error
			return &rt.Error{
				NotFound: false,
			}
		},
	}

	sg := &Sendgrid{RT: mockClient}

	api := rest.NewApi()
	router, err := rest.MakeRouter(
		rest.Post("/sendgrid/mx", sg.ReceiveHandler),
	)
	if err != nil {
		t.Fatal(err)
	}
	api.SetApp(router)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("envelope", `{"to":["test1@example.com","test2@example.com"],"from":"sender@example.com"}`)
	_ = writer.WriteField("email", "Test message")
	writer.Close()

	req, err := http.NewRequest("POST", "/sendgrid/mx", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	recorded := test.RunRequest(t, api.MakeHandler(), req)
	// Should return 503 because of server error
	recorded.CodeIs(503)
}

// TestSendgridHandler is a basic integration test
func TestSendgridHandler(t *testing.T) {
	rtServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("Failed to parse form: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		queue := r.FormValue("queue")
		if queue == "" {
			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, "failure: no queue specified")
			return
		}

		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "RT/4.4.4 200 Ok\n\n# Ticket 123 created.")
	}))
	defer rtServer.Close()

	rtConfig := fmt.Sprintf(`{
		"rt-url": "%s",
		"queues": {
			"test@example.com": "test-queue"
		}
	}`, rtServer.URL)

	tmpfile := "/tmp/rt-mail-test-sendgrid.json"
	if err := os.WriteFile(tmpfile, []byte(rtConfig), 0644); err != nil {
		t.Fatal(err)
	}

	rtClient, err := rt.New(tmpfile)
	if err != nil {
		t.Fatal(err)
	}

	sg := &Sendgrid{RT: rtClient}

	api := rest.NewApi()
	router, err := rest.MakeRouter(
		rest.Post("/sendgrid/mx", sg.ReceiveHandler),
	)
	if err != nil {
		t.Fatal(err)
	}
	api.SetApp(router)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("envelope", `{"to":["test@example.com"],"from":"sender@example.com"}`)
	_ = writer.WriteField("email", "Test message")
	writer.Close()

	req, err := http.NewRequest("POST", "/sendgrid/mx", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	recorded := test.RunRequest(t, api.MakeHandler(), req)
	recorded.CodeIs(204)
}
