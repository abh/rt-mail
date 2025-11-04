package mailgun

import (
	"net/http"

	"github.com/ant0ine/go-json-rest/rest"
	"go.ntppool.org/common/logger"

	"go.askask.com/rt-mail/rt"
)

type Mailgun struct {
	RT *rt.RT
}

func (mg *Mailgun) GetRoutes() []*rest.Route {
	return []*rest.Route{
		rest.Post("/mg/mx/mime", mg.ReceiveHandler),
	}
}

func (mg *Mailgun) ReceiveHandler(w rest.ResponseWriter, r *rest.Request) {
	ctx := r.Request.Context()
	log := logger.FromContext(ctx)

	log.DebugContext(ctx, "received POST request",
		"path", r.URL.String(),
		"content_type", r.Header.Get("Content-Type"),
	)

	r.Body = http.MaxBytesReader(w.(http.ResponseWriter), r.Body, 1024*1024*50)
	defer r.Body.Close()
	r.ParseMultipartForm(64 << 20)

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

	err := mg.RT.Postmail(ctx, recipient, body)
	if err != nil {
		log.ErrorContext(ctx, "failed to post to RT", "error", err, "recipient", recipient)
		if err, ok := err.(*rt.Error); ok {
			if err.NotFound {
				w.WriteHeader(404)
				return
			}
		}
		w.WriteHeader(503)
		return
	}

	log.InfoContext(ctx, "successfully posted to RT", "recipient", recipient)
	w.WriteHeader(204)
}
