package sendgrid

import (
	"github.com/ant0ine/go-json-rest/rest"

	"go.askask.com/rt-mail/rt"
	"github.com/sendgrid/sendgrid-go/helpers/inbound"
)

type Sendgrid struct {
	RT *rt.RT
}

func (sg *Sendgrid) GetRoutes() []*rest.Route {
	return []*rest.Route{
		// rest.Head("/sendgrid", headHandler),
		rest.Post("/sendgrid/mx", sg.ReceiveHandler),
	}
}

func (sg *Sendgrid) ReceiveHandler(w rest.ResponseWriter, r *rest.Request) {
	fmt.Printf("POST to '%s': %#v\n\n", r.URL.String(), r)

	parsedEmail, err := Parse(request)
	if err != nil {
		fmt.Printf("Could not parse email: %s", err)
		w.WriteHeader(400)
		return
	}

	fmt.Printf("Got a message from '%s' to '%s'", parsedEmail.Headers["From"], parsedEmail.Headers["To"])
    
	err = sp.RT.Postmail(parsedEmail.Headers["To"], parsedEmail.rawRequest)
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
