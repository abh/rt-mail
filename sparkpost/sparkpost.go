package sparkpost

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"go.askask.com/rt-mail/rt"
	"go.ntppool.org/common/logger"

	sparkevents "github.com/SparkPost/gosparkpost/events"
)

type SparkPost struct {
	RT *rt.RT
}

func (sp *SparkPost) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/spark", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			headHandler(w, r)
		} else if r.Method == http.MethodPost {
			sp.EventHandler(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/spark/mx", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			sp.RelayHandler(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

func (sp *SparkPost) EventHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	log := logger.FromContext(ctx)

	log.DebugContext(ctx, "received POST request", "path", r.URL.String())

	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024*50)
	defer r.Body.Close()
	r.ParseMultipartForm(64 << 20)

	if r.URL.Path == "/spark/mx" {
		msg, err := io.ReadAll(r.Body)
		if err != nil {
			log.ErrorContext(ctx, "failed to read body", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		log.DebugContext(ctx, "received body", "size", len(msg))
		w.WriteHeader(http.StatusOK)
		return
	}

	var evts sparkevents.Events
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&evts)
	if err != nil {
		log.ErrorContext(ctx, "failed to parse JSON", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	for _, e := range evts {
		log.InfoContext(ctx, "received event", "type", e.EventType())
		if el, ok := e.(sparkevents.ECLogger); ok {
			el.ECLog()
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (sp *SparkPost) RelayHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	log := logger.FromContext(ctx)

	log.DebugContext(ctx, "received POST request", "path", r.URL.String())

	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024*50)
	defer r.Body.Close()
	r.ParseMultipartForm(64 << 20)

	var msgWrapper []struct {
		Msys map[string]json.RawMessage `json:"msys"`
	}

	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&msgWrapper)
	if err != nil {
		log.ErrorContext(ctx, "failed to parse JSON", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var msgs []sparkevents.RelayMessage

	for _, wrapper := range msgWrapper {
		for _, rawMsg := range wrapper.Msys {
			msg := sparkevents.RelayMessage{}
			err = json.Unmarshal(rawMsg, &msg)
			if err != nil {
				log.ErrorContext(ctx, "failed to decode message", "error", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			msgs = append(msgs, msg)
		}
	}

	for _, m := range msgs {
		log.InfoContext(ctx, "processing relay message",
			"from", m.From,
			"to", m.To,
			slog.Group("message",
				"subject", m.Content.Subject,
			),
		)

		err = sp.RT.Postmail(m.To, m.Content.Email)
		if err != nil {
			log.ErrorContext(ctx, "failed to post to RT", "error", err, "recipient", m.To)
			if err, ok := err.(*rt.Error); ok {
				if err.NotFound {
					w.WriteHeader(http.StatusNotFound)
					return
				}
			}
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

func headHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	log := logger.FromContext(ctx)

	log.DebugContext(ctx, "received HEAD request", "path", r.URL.String())
	w.WriteHeader(http.StatusOK)
}
