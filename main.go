package main

import (
	"context"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/docgen"
	"github.com/go-chi/render"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sksmith/smfg-inventory/admin"
	"github.com/sksmith/smfg-inventory/api"
	"github.com/sksmith/smfg-inventory/db"
	"github.com/sksmith/smfg-inventory/inventory"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

)

var dbPool *pgxpool.Pool
var config *AppConfig

func main() {
	ctx := context.Background()

	var err error
	config, err = LoadConfigs()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load configurations")
	}
	configLogging()

	configDatabase(ctx)
	r := configureRouter()

	if config.GenerateRoutes {
		createRouteDocs(r)
	}

	log.Info().Str("port", config.Port).Msg("listening")
	log.Fatal().Err(http.ListenAndServe(":" + config.Port, r))
}

func configDatabase(ctx context.Context) {
	if !config.InMemoryDb {
		var err error

		log.Info().Msg("executing migrations")

		if err = db.RunMigrations(
			config.DbHost,
			config.DbName,
			config.DbPort,
			config.DbUser,
			config.DbPass); err != nil {
			log.Warn().Err(err).Msg("error executing migrations")
		}

		connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			config.DbHost, config.DbPort, config.DbUser, config.DbPass, config.DbName)

		for {
			dbPool, err = db.ConnectDb(ctx, connStr)
			if err != nil {
				log.Error().Err(err).Msg("failed to create connection pool... retrying")
				time.Sleep(1 * time.Second)
				continue
			}
			break
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

	if config.InMemoryDb {
		repo = inventory.NewMemoryRepo()
	} else {
		repo = inventory.NewPostgresRepo(dbPool)
	}

	var queue inventory.Queue
	var err error

	for {
		queue, err = inventory.NewRabbitClient(
			config.QName,
			config.QUser,
			config.QPass,
			config.QHost,
			config.QPort)
		if err != nil {
			log.Error().Err(err).Msg("failed to connect to rabbitmq... retrying")
			time.Sleep(1 * time.Second)
			continue
		}
		break
	}

	service := inventory.NewService(repo, queue)

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
	if config.LogText {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	level, err := zerolog.ParseLevel(config.LogLevel)
	if err != nil {
		log.Warn().Str("loglevel", config.LogLevel).Err(err).Msg("defaulting to info")
		level = zerolog.InfoLevel
	}
	log.Info().Str("loglevel", level.String()).Msg("setting log level")
	zerolog.SetGlobalLevel(level)
}