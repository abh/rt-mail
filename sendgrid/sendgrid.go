package sendgrid

import (
	"net/http"
	"fmt"
	"github.com/ant0ine/go-json-rest/rest"
	"go.askask.com/rt-mail/rt"
	"encoding/json"
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
	From    string
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

	var result Envelope

	json.Unmarshal([]byte(to), &result)

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
