package main // go.askask.com/rt-mail

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/ant0ine/go-json-rest/rest"
	"go.ntppool.org/common/logger"

	"go.askask.com/rt-mail/mailgun"
	requesttracker "go.askask.com/rt-mail/rt"
	"go.askask.com/rt-mail/sendgrid"
	"go.askask.com/rt-mail/sparkpost"
)

var (
	configfile = flag.String("config", "rt-mail.json", "pathname of JSON configuration file")
	listen     = flag.String("listen", ":8002", "listen address")
)

// LoggerMiddleware injects the configured logger into the request context
type LoggerMiddleware struct {
	Logger *slog.Logger
}

// MiddlewareFunc implements the rest.Middleware interface
func (lm *LoggerMiddleware) MiddlewareFunc(handler rest.HandlerFunc) rest.HandlerFunc {
	return func(w rest.ResponseWriter, r *rest.Request) {
		// Inject logger into the request context
		ctx := logger.NewContext(r.Request.Context(), lm.Logger)
		r.Request = r.Request.WithContext(ctx)
		handler(w, r)
	}
}

func newAPI(log *slog.Logger) *rest.Api {
	api := rest.NewApi()
	api.Use(
		&LoggerMiddleware{Logger: log},
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

	// Initialize structured logger
	log := logger.Setup()
	ctx := logger.NewContext(context.Background(), log)

	rt, err := requesttracker.New(*configfile)
	if err != nil {
		log.ErrorContext(ctx, "failed to setup RT interface", "error", err)
		os.Exit(1)
	}

	api := newAPI(log)

	spark := &sparkpost.SparkPost{RT: rt}
	sg := &sendgrid.Sendgrid{RT: rt}
	mg := &mailgun.Mailgun{RT: rt}

	providers := []provider{
		spark, sg, mg,
	}

	routes := make([]*rest.Route, 0)
	for _, p := range providers {
		routes = append(routes, p.GetRoutes()...)
	}

	router, err := rest.MakeRouter(
		routes...,
	)
	if err != nil {
		log.ErrorContext(ctx, "failed to create router", "error", err)
		os.Exit(1)
	}
	api.SetApp(router)

	mux := http.NewServeMux()
	mux.Handle("/", api.MakeHandler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(204)
		return
	})

	log.InfoContext(ctx, "starting server", "listen", *listen)
	if err := http.ListenAndServe(*listen, mux); err != nil {
		log.ErrorContext(ctx, "server error", "error", err)
		os.Exit(1)
	}
}
