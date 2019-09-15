package mailgun

import (
	"fmt"
	"log"
	"net/http"

	"github.com/ant0ine/go-json-rest/rest"

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

	fmt.Printf("POST to '%s': %#v\n\n", r.URL.String(), r)
	fmt.Printf("Content-Type: %s", r.Header.Get("Content-Type"))

	r.Body = http.MaxBytesReader(w.(http.ResponseWriter), r.Body, 1024*1024*50)
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
				w.WriteHeader(404)
				return
			}
		}
		w.WriteHeader(503)
		return
	}

	w.WriteHeader(204)
	// w.WriteJson(struct{ Error string }{Error: "not implemented"})
}
