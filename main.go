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
const (
	AppName = "smfg-inventory"
)
var (
	AppVersion string
	Sha1Version string
	BuildTime string

	dbPool *pgxpool.Pool
	config *AppConfig

	configUrl = os.Getenv("SMFG_CONFIG_SERVER_URL")
	configBranch = os.Getenv("SMFG_CONFIG_SERVER_BRANCH")
	profile = os.Getenv("SMFG_PROFILE")
	
)

func main() {
	ctx := context.Background()

	var err error
	config, err = LoadConfigs(configUrl, configBranch, profile)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load configurations")
	}
	configLogging()
	printLogHeader(config)

	configDatabase(ctx)
	r := configureRouter()

	if config.GenerateRoutes {
		createRouteDocs(r)
	}

	log.Info().Str("port", config.Port).Msg("listening")
	log.Fatal().Err(http.ListenAndServe(":" + config.Port, r))
}

func printLogHeader(c *AppConfig) {
	if c.LogText {
		log.Info().Msg("=============================================")
		log.Info().Msg(fmt.Sprintf("    Application: %s", AppName))
		log.Info().Msg(fmt.Sprintf("        Profile: %s", profile))
		log.Info().Msg(fmt.Sprintf("  Config Server: %s - %s", configUrl, configBranch))
		log.Info().Msg(fmt.Sprintf("    Tag Version: %s", AppVersion))
		log.Info().Msg(fmt.Sprintf("   Sha1 Version: %s", Sha1Version))
		log.Info().Msg(fmt.Sprintf("     Build Time: %s", BuildTime))
		log.Info().Msg("=============================================")
	} else {
		log.Info().Str("application", AppName).
			Str("version", AppVersion).
			Str("sha1ver", Sha1Version).
			Str("build-time", BuildTime).
			Str("profile", profile).
			Str("config-url", configUrl).
			Str("config-branch", configBranch)
	}
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