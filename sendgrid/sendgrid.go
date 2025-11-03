package sendgrid

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/ant0ine/go-json-rest/rest"
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
	fmt.Printf("POST to '%s': %#v\n\n", r.URL.String(), r)

	r.Body = http.MaxBytesReader(w.(http.ResponseWriter), r.Body, 1024*1024*50)
	defer r.Body.Close()
	r.ParseMultipartForm(64 << 20)

	form := r.PostForm
	// for k, v := range form {
	// 	fmt.Printf("data %s: %s \n", k, v)
	// }

	to := form.Get("envelope")
	if to == "" {
		log.Printf("Missing envelope field")
		w.WriteHeader(400)
		return
	}

	var result Envelope
	if err := json.Unmarshal([]byte(to), &result); err != nil {
		log.Printf("Failed to parse envelope: %s", err)
		w.WriteHeader(400)
		return
	}

	if len(result.To) == 0 {
		log.Printf("Envelope contains no recipients")
		w.WriteHeader(400)
		return
	}

	fmt.Printf("envelope.to %s: \n", result.To)

	body := form.Get("email")

	allNotFound := true
	var err error

	//fmt.Printf("body: %s \n", body)

	for _, email := range result.To {
		fmt.Printf("to: %s \n", email)
		err = sg.RT.Postmail(email, body)
		if err != nil {
			fmt.Printf("post error: %s \n", err)
			if err, ok := err.(*rt.Error); ok {
				if err.NotFound {
					fmt.Printf("inserting email %s failed, address not setup\n", email)
					continue
				}
				w.WriteHeader(503)
				break
			}
			continue
		}
		fmt.Printf("inserting email succeeded %s\n", email)
		allNotFound = false
	}

	if allNotFound == true {
		w.WriteHeader(404)
		return
	}

	w.WriteHeader(204)
}
