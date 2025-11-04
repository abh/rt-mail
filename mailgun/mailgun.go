package mailgun

import (
	"fmt"
	"log"
	"net/http"

	"go.askask.com/rt-mail/rt"
)

type Mailgun struct {
	RT *rt.RT
}

func (mg *Mailgun) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/mg/mx/mime", mg.ReceiveHandler)
}

func (mg *Mailgun) ReceiveHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	fmt.Printf("POST to '%s': %#v\n\n", r.URL.String(), r)
	fmt.Printf("Content-Type: %s", r.Header.Get("Content-Type"))

	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024*50)
	defer r.Body.Close()
	r.ParseMultipartForm(64 << 20)

	form := r.PostForm
	for k, v := range form {
		log.Printf("data %s: %s", k, v)
	}

	recipient := form.Get("recipient")
	body := form.Get("body-mime")

	err := mg.RT.Postmail(recipient, body)
	if err != nil {
		log.Printf("post error: %s", err)
		if err, ok := err.(*rt.Error); ok {
			if err.NotFound {
				w.WriteHeader(http.StatusNotFound)
				return
			}
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
