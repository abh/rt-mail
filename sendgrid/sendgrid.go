package sendgrid

import (
	"encoding/json"
	"net/http"

	"github.com/ant0ine/go-json-rest/rest"
	"go.ntppool.org/common/logger"

	"go.askask.com/rt-mail/rt"
)

type Sendgrid struct {
	RT *rt.RT
}

func (sg *Sendgrid) GetRoutes() []*rest.Route {
	return []*rest.Route{
		rest.Post("/sendgrid/mx", sg.ReceiveHandler),
	}
}

type Envelope struct {
	From string
	To   []string
}

func (sg *Sendgrid) ReceiveHandler(w rest.ResponseWriter, r *rest.Request) {
	ctx := r.Request.Context()
	log := logger.FromContext(ctx)

	log.DebugContext(ctx, "received POST request", "path", r.URL.String())

	r.Body = http.MaxBytesReader(w.(http.ResponseWriter), r.Body, 1024*1024*50)
	defer r.Body.Close()
	r.ParseMultipartForm(64 << 20)

	form := r.PostForm
	envelopeData := form.Get("envelope")

	var envelope Envelope
	if err := json.Unmarshal([]byte(envelopeData), &envelope); err != nil {
		log.ErrorContext(ctx, "failed to parse envelope", "error", err)
		w.WriteHeader(400)
		return
	}

	if len(envelope.To) == 0 {
		log.ErrorContext(ctx, "envelope contains no recipients")
		w.WriteHeader(400)
		return
	}

	log.DebugContext(ctx, "parsed envelope",
		"from", envelope.From,
		"to", envelope.To,
	)

	body := form.Get("email")
	if body == "" {
		log.ErrorContext(ctx, "email body is empty")
		w.WriteHeader(400)
		return
	}

	allNotFound := true

	for _, email := range envelope.To {
		log.InfoContext(ctx, "processing sendgrid webhook", "recipient", email)

		err := sg.RT.Postmail(ctx, email, body)
		if err != nil {
			log.ErrorContext(ctx, "failed to post to RT", "error", err, "recipient", email)
			if err, ok := err.(*rt.Error); ok {
				if err.NotFound {
					log.WarnContext(ctx, "recipient address not configured", "recipient", email)
					continue
				}
				w.WriteHeader(503)
				break
			}
			continue
		}

		log.InfoContext(ctx, "successfully posted to RT", "recipient", email)
		allNotFound = false
	}

	if allNotFound {
		log.WarnContext(ctx, "all recipients were not found")
		w.WriteHeader(404)
		return
	}

	w.WriteHeader(204)
}
