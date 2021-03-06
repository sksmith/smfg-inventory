package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/docgen"
	"github.com/go-chi/render"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sksmith/bunnyq"
	"github.com/sksmith/smfg-inventory/admin"
	"github.com/sksmith/smfg-inventory/api"
	"github.com/sksmith/smfg-inventory/db"
	"github.com/sksmith/smfg-inventory/inventory"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	AppName = "smfg-inventory"
)

var (
	AppVersion  string
	Sha1Version string
	BuildTime   string

	dbPool *pgxpool.Pool
	config *AppConfig

	configUrl    = os.Getenv("SMFG_CONFIG_SERVER_URL")
	configBranch = os.Getenv("SMFG_CONFIG_SERVER_BRANCH")
	profile      = os.Getenv("SMFG_PROFILE")
)

func main() {
	ctx := context.Background()

	var err error
	log.Info().Msg("loading configurations...")
	config, err = LoadConfigs(configUrl, configBranch, profile)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load configurations")
	}
	log.Info().Msg("configuring logging...")
	configLogging()

	config.Revision = "2"
	printLogHeader(config)

	log.Info().Msg("connecting to the database...")
	configDatabase(ctx)
	repo := inventory.NewPostgresRepo(dbPool)

	log.Info().Msg("connecting to rabbitmq...")
	queue := rabbit()

	log.Info().Msg("configuring router...")
	r := configureRouter(queue, repo, config.QInventoryExchange, config.QReservationExchange)

	log.Info().Msg("generating configurations...")
	if config.GenerateRoutes {
		createRouteDocs(r)
	}

	log.Info().Str("port", config.Port).Msg("listening")
	log.Fatal().Err(http.ListenAndServe(":"+config.Port, r))
}

func rabbit() inventory.Queue {
	var queue inventory.Queue
	osChannel := make(chan os.Signal, 1)
	signal.Notify(osChannel, os.Kill)

	for {
		queue = bunnyq.New(context.Background(),
			bunnyq.Address{
				User: config.QUser,
				Pass: config.QPass,
				Host: config.QHost,
				Port: config.QPort,
			},
			osChannel,
			bunnyq.LogHandler(logger{}),
		)

		break
	}

	return queue
}

type logger struct {
}

func (l logger) Log(ctx context.Context, level bunnyq.LogLevel, msg string, data map[string]interface{}) {
	var evt *zerolog.Event
	switch level {
	case bunnyq.LogLevelTrace:
		evt = log.Trace()
	case bunnyq.LogLevelDebug:
		evt = log.Debug()
	case bunnyq.LogLevelInfo:
		evt = log.Info()
	case bunnyq.LogLevelWarn:
		evt = log.Warn()
	case bunnyq.LogLevelError:
		evt = log.Error()
	case bunnyq.LogLevelNone:
		evt = log.Info()
	default:
		evt = log.Info()
	}

	for k, v := range data {
		evt.Interface(k, v)
	}

	evt.Msg(msg)
}

func printLogHeader(c *AppConfig) {
	if c.LogText {
		log.Info().Msg("=============================================")
		log.Info().Msg(fmt.Sprintf("    Application: %s", AppName))
		log.Info().Msg(fmt.Sprintf("       Revision: %s", c.Revision))
		log.Info().Msg(fmt.Sprintf("        Profile: %s", profile))
		log.Info().Msg(fmt.Sprintf("  Config Server: %s - %s", configUrl, configBranch))
		log.Info().Msg(fmt.Sprintf("    Tag Version: %s", AppVersion))
		log.Info().Msg(fmt.Sprintf("   Sha1 Version: %s", Sha1Version))
		log.Info().Msg(fmt.Sprintf("     Build Time: %s", BuildTime))
		log.Info().Msg("=============================================")
	} else {
		log.Info().Str("application", AppName).
			Str("revision", c.Revision).
			Str("version", AppVersion).
			Str("sha1ver", Sha1Version).
			Str("build-time", BuildTime).
			Str("profile", profile).
			Str("config-url", configUrl).
			Str("config-branch", configBranch).
			Send()
	}
}

func configDatabase(ctx context.Context) {
	if !config.InMemoryDb {
		var err error

		if config.DbMigrate {
			log.Info().Msg("executing migrations")

			if err = db.RunMigrations(
				config.DbHost,
				config.DbName,
				config.DbPort,
				config.DbUser,
				config.DbPass); err != nil {
				log.Warn().Err(err).Msg("error executing migrations")
			}
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

func configureRouter(queue inventory.Queue, repo inventory.Repository, invExchange, resExchange string) chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.URLFormat)
	r.Use(api.MetricsMiddleware)
	r.Use(render.SetContentType(render.ContentTypeJSON))
	r.Use(api.LoggingMiddleware)

	r.Handle("/inventory/metrics", promhttp.Handler())
	r.Route("/inventory/v1", inventoryApi(queue, repo, invExchange, resExchange))
	r.Mount("/inventory/admin", admin.Router())

	return r
}

func inventoryApi(queue inventory.Queue, repo inventory.Repository, invExchange, resExchange string) func(r chi.Router) {
	return func(r chi.Router) {
		service := inventory.NewService(repo, queue, invExchange, resExchange)
		invApi := inventory.NewApi(service)
		invApi.ConfigureRouter(r)
	}
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
