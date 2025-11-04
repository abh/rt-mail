package main // go.askask.com/rt-mail

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"go.ntppool.org/common/logger"

	"go.askask.com/rt-mail/mailgun"
	"go.askask.com/rt-mail/middleware"
	requesttracker "go.askask.com/rt-mail/rt"
	"go.askask.com/rt-mail/sendgrid"
	"go.askask.com/rt-mail/ses"
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

	// Initialize structured logger
	log := logger.Setup()
	ctx := logger.NewContext(context.Background(), log)

	rt, err := requesttracker.New(*configfile)
	if err != nil {
		log.ErrorContext(ctx, "failed to setup RT interface", "error", err)
		os.Exit(1)
	}

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
			log.ErrorContext(ctx, "failed to setup SES handler", "error", err)
			os.Exit(1)
		}
		providers = append(providers, sesHandler)
		log.InfoContext(ctx, "SES handler enabled", "topic_arn", topicARN)
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

	log.InfoContext(ctx, "starting server", "listen", *listen)
	if err := http.ListenAndServe(*listen, handler); err != nil {
		log.ErrorContext(ctx, "server error", "error", err)
		os.Exit(1)
	}
}
