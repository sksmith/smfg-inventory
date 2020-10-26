package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/docgen"
	"github.com/go-chi/render"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rs/zerolog"
	"github.com/sksmith/smfg-inventory/admin"
	"github.com/sksmith/smfg-inventory/api"
	"github.com/sksmith/smfg-inventory/db"
	"github.com/sksmith/smfg-inventory/inventory"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

var routes = flag.Bool("routes", false, "Generate router documentation")
var port = flag.String("port", "8080", "Port the application should listen on")
var logLevel = flag.String("logLevel", "trace", "The minimum level for logs to print")
var inMemory = flag.Bool("inMemory", false, "Start the application using an in memory db for dev")
var textLogging = flag.Bool("textLogging", false, "Log using text output rather than structured")

var dbpool *pgxpool.Pool

func main() {
	ctx := context.Background()

	flag.Parse()
	configLogging()

	configDatabase(ctx)
	r := configureRouter()

	if *routes {
		createRouteDocs(r)
	}

	log.Info().Str("port", *port).Msg("listening")
	log.Fatal().Err(http.ListenAndServe(":" + *port, r))
}

func configDatabase(ctx context.Context) {
	if !*inMemory {
		var err error

		host := os.Getenv("DB_HOST")
		port := os.Getenv("DB_PORT")
		user := os.Getenv("DB_USER")
		pass := os.Getenv("DB_PASS")
		name := os.Getenv("DB_NAME")

		log.Info().Msg("executing migrations")


		if err = db.RunMigrations(host, name, port, user, pass); err != nil {
			log.Fatal().Err(err).Msg("error executing migrations")
		}

		connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			host, port, user, pass, name)

		dbpool, err = db.ConnectDb(ctx, connStr)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to create connection pool")
			os.Exit(1)
		}
	}
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
	var repo inventory.Repository

	if *inMemory {
		repo = inventory.NewMemoryRepo()
	} else {
		repo = inventory.NewPostgresRepo(dbpool)
	}

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

func configLogging() {
	if *textLogging {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	level, err := zerolog.ParseLevel(*logLevel)
	if err != nil {
		log.Warn().Str("loglevel", *logLevel).Err(err).Msg("defaulting to info")
		level = zerolog.InfoLevel
	}
	log.Info().Str("loglevel", level.String()).Msg("setting log level")
	zerolog.SetGlobalLevel(level)
}