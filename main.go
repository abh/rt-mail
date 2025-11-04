package main // go.askask.com/rt-mail

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"go.askask.com/rt-mail/mailgun"
	"go.askask.com/rt-mail/middleware"
	requesttracker "go.askask.com/rt-mail/rt"
	"go.askask.com/rt-mail/sendgrid"
	"go.askask.com/rt-mail/sparkpost"
)

var (
	configfile = flag.String("config", "rt-mail.json", "pathname of JSON configuration file")
	listen     = flag.String("listen", ":8002", "listen address")
)

func init() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: rt-mail -config=rt-mail.json -listen=:8080")
		flag.PrintDefaults()
	}
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Lshortfile)
}

type provider interface {
	RegisterRoutes(mux *http.ServeMux)
}

func main() {
	flag.Parse()

	rt, err := requesttracker.New(*configfile)
	if err != nil {
		log.Fatalf("setting up RT interface: %s", err)
	}

	spark := &sparkpost.SparkPost{RT: rt}
	sg := &sendgrid.Sendgrid{RT: rt}
	mg := &mailgun.Mailgun{RT: rt}

	providers := []provider{
		spark, sg, mg,
	}

	mux := http.NewServeMux()

	// Register all provider routes
	for _, p := range providers {
		p.RegisterRoutes(mux)
	}

	// Add healthz endpoint
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	// Apply middleware
	handler := middleware.Chain(mux,
		middleware.Recovery,
		middleware.Logging,
		middleware.Gzip,
	)

	log.Printf("Listening on '%s'", *listen)
	log.Fatal(http.ListenAndServe(*listen, handler))
}
