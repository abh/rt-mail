package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	sparkevents "github.com/SparkPost/gosparkpost/events"
	"github.com/ant0ine/go-json-rest/rest"
)

type MandrillMsg struct {
	RawMsg    string                 `json:"raw_msg"`
	Headers   map[string]interface{} `json:"headers"`
	Text      string                 `json:"text"`
	Email     string                 `json:"email"`
	FromEmail string                 `json:"from_email"`
	Subject   string                 `json:"subject"`
}

type MandrillEvent struct {
	Event string      `json:"event"`
	Msg   MandrillMsg `json:"msg"`
}

// Address to Queue configuration
type AddressQueue map[string]string

type Config struct {
	RTUrl  string       `json:"rt-url"`
	Queues AddressQueue `json:"queues"`
}

var (
	configfile = flag.String("config", "sparkpost-rt.json", "pathname of JSON configuration file")
	listen     = flag.String("listen", ":8002", "listen address")

	mux *http.ServeMux

	config Config
)

var Version string

func eventHandler(w rest.ResponseWriter, r *rest.Request) {

	fmt.Printf("POST to '%s': %#v\n\n", r.URL.String(), r)

	r.Body = http.MaxBytesReader(w.(http.ResponseWriter), r.Body, 1024*1024*50)
	defer r.Body.Close()
	r.ParseMultipartForm(64 << 20)

	if r.URL.Path == "/spark/mx" {
		msg, err := ioutil.ReadAll(r.Body)
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

func relayHandler(w rest.ResponseWriter, r *rest.Request) {

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

		queue, action := addressToQueueAction(m.To)

		form := url.Values{
			"queue":  []string{queue},
			"action": []string{action},
		}
		log.Printf("posting to queue '%s' (action: '%s')", queue, action)

		form.Add("message", m.Content.Email)

		resp, err := http.PostForm(
			config.RTUrl,
			form,
		)
		if err != nil {
			log.Println("PostForm err:", err)
			w.WriteHeader(500)
			return
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println("Error reading RT response: ", err)
			w.WriteHeader(500)
			return
		}
		resp.Body.Close()
		log.Println("RT Response: ", string(body))

		if resp.StatusCode > 299 {
			w.WriteHeader(503)
			return
		}
	}

	w.WriteHeader(204)

}

func newAPI() *rest.Api {
	api := rest.NewApi()
	api.Use(
		&rest.AccessLogApacheMiddleware{
			Format: rest.CombinedLogFormat,
		},
		&rest.TimerMiddleware{},
		&rest.RecorderMiddleware{},
		&rest.RecoverMiddleware{},
		&rest.GzipMiddleware{},
		// &rest.ContentTypeCheckerMiddleware{},
	)

	router, err := rest.MakeRouter(
		rest.Head("/spark", headHandler),
		rest.Post("/spark", eventHandler),
		rest.Post("/spark/mx", relayHandler),
	)
	if err != nil {
		log.Fatal(err)
	}
	api.SetApp(router)
	return api
}

func init() {

	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: example [-cert=<cert>] [-key=<key>] [-config=<config>] [-listen=<listen>]")
		flag.PrintDefaults()
	}
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Lshortfile)

	api := newAPI()
	mux = http.NewServeMux()
	mux.Handle("/", api.MakeHandler())
}

func loadConfig(file string) error {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	err = json.Unmarshal(b, &config)
	if err != nil {
		return err
	}

	return nil
}

func addressToQueueAction(email string) (string, string) {

	email = strings.ToLower(email)

	idx := strings.Index(email, "@")
	if idx < 1 {
		return "", "correspond"
	}

	local := email[0:idx]

	for _, address := range []string{email, local} {
		for target, queue := range config.Queues {
			// log.Printf("testing address address='%s' target='%s' queue='%s'",
			// 	address, target, queue)

			if address == target {
				return queue, "correspond"
			}
			if idx = strings.Index(target, "@"); idx > 0 {
				target = target[0:idx] + "-comment" + target[idx:]
			} else {
				target = target + "-comment"
			}
			if address == target {
				return queue, "comment"
			}
		}
	}

	return "", "correspond"
}

func main() {
	flag.Parse()

	err := loadConfig(*configfile)
	if err != nil {
		log.Printf("Could not load configuration file '%s': %s", *configfile, err)
	}

	log.Printf("Listening on '%s'", *listen)
	log.Fatal(http.ListenAndServe(*listen, mux))
}

func headHandler(w rest.ResponseWriter, r *rest.Request) {
	fmt.Printf("HEAD for %sv\n", r.URL.String())
	w.WriteHeader(200)
}
