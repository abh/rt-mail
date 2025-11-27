package mailgun

import (
	"net/http"

	"go.ntppool.org/common/logger"

	"go.askask.com/rt-mail/rt"
)

type Mailgun struct {
	RT rt.Client
}

func (mg *Mailgun) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/mg/mx/mime", mg.ReceiveHandler)
}

func (mg *Mailgun) ReceiveHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	log := logger.FromContext(ctx)

	log.DebugContext(ctx, "received POST request",
		"path", r.URL.String(),
		"content_type", r.Header.Get("Content-Type"),
	)

	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024*50)
	defer func() { _ = r.Body.Close() }()
	_ = r.ParseMultipartForm(64 << 20)

	form := r.PostForm

	// Log form fields at debug level (excluding sensitive content)
	var formKeys []string
	for k := range form {
		formKeys = append(formKeys, k)
	}
	log.DebugContext(ctx, "parsed form data", "fields", formKeys)

	recipient := form.Get("recipient")
	body := form.Get("body-mime")

	log.InfoContext(ctx, "processing mailgun webhook", "recipient", recipient)

	err := mg.RT.Postmail(recipient, body)
	if err != nil {
		log.ErrorContext(ctx, "failed to post to RT", "error", err, "recipient", recipient)
		if err, ok := err.(*rt.Error); ok {
			if err.NotFound {
				w.WriteHeader(http.StatusNotFound)
				return
			}
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	log.InfoContext(ctx, "successfully posted to RT", "recipient", recipient)
	w.WriteHeader(http.StatusNoContent)
}
