package main // go.askask.com/rt-mail

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	// _ "go.askask.com/rt-mail/mailgun"

	"github.com/ant0ine/go-json-rest/rest"

	"go.askask.com/rt-mail/mailgun"
	requesttracker "go.askask.com/rt-mail/rt"
	"go.askask.com/rt-mail/sendgrid"
	"go.askask.com/rt-mail/ses"
	"go.askask.com/rt-mail/sparkpost"
)

var (
	configfile = flag.String("config", "rt-mail.json", "pathname of JSON configuration file")
	listen     = flag.String("listen", ":8002", "listen address")
)

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
	return api
}

func init() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: rt-mail -config=rt-mail.json -listen=:8080")
		flag.PrintDefaults()
	}
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Lshortfile)
}

type provider interface {
	GetRoutes() []*rest.Route
}

func main() {
	flag.Parse()

	rt, err := requesttracker.New(*configfile)
	if err != nil {
		log.Fatalf("setting up RT interface: %s", err)
	}

	api := newAPI()

	spark := &sparkpost.SparkPost{RT: rt}
	sg := &sendgrid.Sendgrid{RT: rt}
	mg := &mailgun.Mailgun{RT: rt}

	providers := []provider{
		spark, sg, mg,
	}

	// Add SES provider if configured
	if topicARN := os.Getenv("RT_SES_SNS_TOPIC_ARN"); topicARN != "" {
		sesHandler, err := ses.New(rt, topicARN)
		if err != nil {
			log.Fatalf("setting up SES handler: %s", err)
		}
		providers = append(providers, sesHandler)
		log.Printf("SES handler enabled for topic %s", topicARN)
	}

	routes := make([]*rest.Route, 0)
	for _, p := range providers {
		routes = append(routes, p.GetRoutes()...)
	}

	router, err := rest.MakeRouter(
		routes...,
	)
	if err != nil {
		log.Fatal(err)
	}
	api.SetApp(router)

	mux := http.NewServeMux()
	mux.Handle("/", api.MakeHandler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(204)
		return
	})

	log.Printf("Listening on '%s'", *listen)
	log.Fatal(http.ListenAndServe(*listen, mux))
}
