package sendgrid

import (
	"regexp"
	"net/http"
	"fmt"
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

func (sg *Sendgrid) ReceiveHandler(w rest.ResponseWriter, r *rest.Request) {
	fmt.Printf("POST to '%s': %#v\n\n", r.URL.String(), r)

	r.Body = http.MaxBytesReader(w.(http.ResponseWriter), r.Body, 1024*1024*50)
	defer r.Body.Close()
	r.ParseMultipartForm(64 << 20)

	form := r.PostForm
	// for k, v := range form {
	// 	fmt.Printf("data %s: %s \n", k, v)
	// }

	recipient := form.Get("to")
	re, _ := regexp.Compile("<(.*?)>")
	to := re.FindStringSubmatch(recipient)
	body := form.Get("email")

	toEmail := recipient
	if len(to) >= 2 {
		toEmail = to[1]
	}

	// fmt.Printf("to: %s \n", toEmail)
	// fmt.Printf("body: %s \n", body)

	err := sg.RT.Postmail(toEmail, body)
	if err != nil {
		fmt.Printf("post error: %s", err)
		if err, ok := err.(*rt.Error); ok {
			if err.NotFound {
				w.WriteHeader(404)
				return
			}
		}
		w.WriteHeader(503)
	}

	w.WriteHeader(204)
}
