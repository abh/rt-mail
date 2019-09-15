package sendgrid

import (
	"github.com/ant0ine/go-json-rest/rest"

	"go.askask.com/rt-mail/rt"
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
	w.WriteHeader(500)
	w.WriteJson(struct{ Error string }{Error: "not implemented"})
}
