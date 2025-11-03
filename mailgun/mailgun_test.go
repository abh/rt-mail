package mailgun

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

func TestMailgunReceiveHandler_Success(t *testing.T) {
	// Create mock RT client
	mockClient := &mockRT{
		postmailFunc: func(recipient string, message string) error {
			if recipient == "" {
				t.Error("Expected recipient to be set")
			}
			if message == "" {
				t.Error("Expected message to be set")
			}
			return nil
		},
	}

	// Create mailgun handler with mock RT
	mg := &Mailgun{RT: mockClient}

	// Create API
	api := rest.NewApi()
	router, err := rest.MakeRouter(
		rest.Post("/mg/mx/mime", mg.ReceiveHandler),
	)
	if err != nil {
		t.Fatal(err)
	}
	api.SetApp(router)

	// Create multipart form data
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("recipient", "test@example.com")
	_ = writer.WriteField("body-mime", "From: sender@example.com\nTo: test@example.com\nSubject: Test\n\nTest message")
	writer.Close()

	// Make request
	req, err := http.NewRequest("POST", "/mg/mx/mime", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	recorded := test.RunRequest(t, api.MakeHandler(), req)
	recorded.CodeIs(204)
}

func TestMailgunReceiveHandler_NotFound(t *testing.T) {
	// Create mock RT client that returns NotFound error
	mockClient := &mockRT{
		postmailFunc: func(recipient string, message string) error {
			return &rt.Error{
				NotFound: true,
			}
		},
	}

	mg := &Mailgun{RT: mockClient}

	api := rest.NewApi()
	router, err := rest.MakeRouter(
		rest.Post("/mg/mx/mime", mg.ReceiveHandler),
	)
	if err != nil {
		t.Fatal(err)
	}
	api.SetApp(router)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("recipient", "unknown@example.com")
	_ = writer.WriteField("body-mime", "Test message")
	writer.Close()

	req, err := http.NewRequest("POST", "/mg/mx/mime", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	recorded := test.RunRequest(t, api.MakeHandler(), req)
	recorded.CodeIs(404)
}

func TestMailgunReceiveHandler_ServerError(t *testing.T) {
	// Create mock RT client that returns a server error
	mockClient := &mockRT{
		postmailFunc: func(recipient string, message string) error {
			return fmt.Errorf("RT server error")
		},
	}

	mg := &Mailgun{RT: mockClient}

	api := rest.NewApi()
	router, err := rest.MakeRouter(
		rest.Post("/mg/mx/mime", mg.ReceiveHandler),
	)
	if err != nil {
		t.Fatal(err)
	}
	api.SetApp(router)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("recipient", "test@example.com")
	_ = writer.WriteField("body-mime", "Test message")
	writer.Close()

	req, err := http.NewRequest("POST", "/mg/mx/mime", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	recorded := test.RunRequest(t, api.MakeHandler(), req)
	recorded.CodeIs(503)
}

func TestMailgunReceiveHandler_LargePayload(t *testing.T) {
	mockClient := &mockRT{
		postmailFunc: func(recipient string, message string) error {
			return nil
		},
	}

	mg := &Mailgun{RT: mockClient}

	api := rest.NewApi()
	router, err := rest.MakeRouter(
		rest.Post("/mg/mx/mime", mg.ReceiveHandler),
	)
	if err != nil {
		t.Fatal(err)
	}
	api.SetApp(router)

	// Create a payload that's close to the 50MB limit
	largeMessage := make([]byte, 1024*1024) // 1MB test message
	for i := range largeMessage {
		largeMessage[i] = 'A'
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("recipient", "test@example.com")
	_ = writer.WriteField("body-mime", string(largeMessage))
	writer.Close()

	req, err := http.NewRequest("POST", "/mg/mx/mime", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	recorded := test.RunRequest(t, api.MakeHandler(), req)
	recorded.CodeIs(204)
}

// TestMailgunHandler is a basic integration test
func TestMailgunHandler(t *testing.T) {
	// Create a test HTTP server that simulates RT
	rtServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse form to get recipient
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

	// Create RT client pointing to test server
	rtConfig := fmt.Sprintf(`{
		"rt-url": "%s",
		"queues": {
			"test@example.com": "test-queue"
		}
	}`, rtServer.URL)

	// Write config to temp file
	tmpfile := "/tmp/rt-mail-test-mailgun.json"
	if err := os.WriteFile(tmpfile, []byte(rtConfig), 0644); err != nil {
		t.Fatal(err)
	}

	rtClient, err := rt.New(tmpfile)
	if err != nil {
		t.Fatal(err)
	}

	mg := &Mailgun{RT: rtClient}

	api := rest.NewApi()
	router, err := rest.MakeRouter(
		rest.Post("/mg/mx/mime", mg.ReceiveHandler),
	)
	if err != nil {
		t.Fatal(err)
	}
	api.SetApp(router)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("recipient", "test@example.com")
	_ = writer.WriteField("body-mime", "Test message")
	writer.Close()

	req, err := http.NewRequest("POST", "/mg/mx/mime", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	recorded := test.RunRequest(t, api.MakeHandler(), req)
	recorded.CodeIs(204)
}
