package main

import (
	"flag"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/docgen"
	"github.com/go-chi/render"
	"github.com/rs/zerolog"
	"github.com/sksmith/smfg-inventory/admin"
	"github.com/sksmith/smfg-inventory/api"
	"github.com/sksmith/smfg-inventory/inventory"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

var routes = flag.Bool("routes", false, "Generate router documentation")
var port = flag.String("port", "8080", "Port the application should listen on")
var logLevel = flag.String("loglevel", "trace", "The minimum level for logs to print")

func main() {
	flag.Parse()
	setLogLevel()

	r := configureRouter()

	if *routes {
		createRouteDocs(r)
	}

	log.Info().Str("port", *port).Msg("Listening")
	log.Fatal().Err(http.ListenAndServe(":" + *port, r))
}

func configureRouter() chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.URLFormat)
	r.Use(api.MetricsMiddleware)
	r.Use(render.SetContentType(render.ContentTypeJSON))
	r.Use(api.LoggingMiddleware)

	r.Handle("/metrics", promhttp.Handler())
	r.Route("/inventory/v1", inventoryApi)
	r.Mount("/admin", admin.Router())

	return r
}

func inventoryApi(r chi.Router) {
	repo := inventory.NewMemoryRepo()
	service := inventory.NewService(repo)
	invApi := inventory.NewApi(service)
	invApi.ConfigureRouter(r)
}

func createRouteDocs(r chi.Router) {
	// TODO See how documentation is generated

	fmt.Println(docgen.MarkdownRoutesDoc(r, docgen.MarkdownOpts{
		ProjectPath: "github.com/sksmith/smfg-inventory",
		Intro:       "The generated API documentation for smfg-inventory.",
	}))
	return
}

func setLogLevel() {
	level, err := zerolog.ParseLevel(*logLevel)
	if err != nil {
		log.Warn().Str("loglevel", *logLevel).Err(err).Msg("defaulting to info")
		level = zerolog.InfoLevel
	}
	log.Info().Str("loglevel", level.String()).Msg("setting log level")
	zerolog.SetGlobalLevel(level)
}