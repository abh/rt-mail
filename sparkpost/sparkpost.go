package sparkpost

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"go.askask.com/rt-mail/rt"

	sparkevents "github.com/SparkPost/gosparkpost/events"
	"github.com/ant0ine/go-json-rest/rest"
)

type SparkPost struct {
	RT *rt.RT
}

func (sp *SparkPost) GetRoutes() []*rest.Route {
	return []*rest.Route{
		rest.Head("/spark", headHandler),
		rest.Post("/spark", sp.EventHandler),
		rest.Post("/spark/mx", sp.RelayHandler),
	}
}

func (sp *SparkPost) EventHandler(w rest.ResponseWriter, r *rest.Request) {

	fmt.Printf("POST to '%s': %#v\n\n", r.URL.String(), r)

	r.Body = http.MaxBytesReader(w.(http.ResponseWriter), r.Body, 1024*1024*50)
	defer r.Body.Close()
	r.ParseMultipartForm(64 << 20)

	if r.URL.Path == "/spark/mx" {
		msg, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("Could not read body: %s", err)
			w.WriteHeader(500)
			return
		}
		log.Printf("Got body: %s", msg)
		w.WriteHeader(200)
		return
	}

	var evts sparkevents.Events
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&evts)
	if err != nil {
		log.Printf("Could not parse JSON: %s", err)
		w.WriteHeader(400)
		return

	}

	for _, e := range evts {
		log.Printf("Got an event of type: %s", e.EventType())
		if el, ok := e.(sparkevents.ECLogger); ok {
			el.ECLog()
		}
	}

	w.WriteHeader(200)
}

func (sp *SparkPost) RelayHandler(w rest.ResponseWriter, r *rest.Request) {

	fmt.Printf("POST to '%s': %#v\n\n", r.URL.String(), r)

	r.Body = http.MaxBytesReader(w.(http.ResponseWriter), r.Body, 1024*1024*50)
	defer r.Body.Close()
	r.ParseMultipartForm(64 << 20)

	var msgWrapper []struct {
		Msys map[string]json.RawMessage `json:"msys"`
	}

	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&msgWrapper)
	if err != nil {
		log.Printf("Could not parse JSON: %s", err)
		w.WriteHeader(400)
		return
	}

	var msgs []sparkevents.RelayMessage

	for _, wrapper := range msgWrapper {
		for _, rawMsg := range wrapper.Msys {
			msg := sparkevents.RelayMessage{}
			err = json.Unmarshal(rawMsg, &msg)
			if err != nil {
				log.Printf("Could not decode raw to msg: %s", err)
				w.WriteHeader(500)
				return
			}
			msgs = append(msgs, msg)
		}
	}

	for _, m := range msgs {
		log.Printf("Got a message from '%s' to '%s': %s", m.From, m.To, m.String())

		js, err := json.MarshalIndent(m, "", "    ")
		if err != nil {
			log.Println("Could not marshall msg to json")
		} else {
			log.Printf("Json:\n%s", string(js))
		}

		err = sp.RT.Postmail(m.To, m.Content.Email)
		if err != nil {
			log.Printf("post error: %s", err)
			if err, ok := err.(*rt.Error); ok {
				if err.NotFound {
					w.WriteHeader(404)
					return
				}
			}
			w.WriteHeader(503)
		}
	}

	w.WriteHeader(204)

}

func headHandler(w rest.ResponseWriter, r *rest.Request) {
	fmt.Printf("HEAD for %sv\n", r.URL.String())
	w.WriteHeader(200)
}
