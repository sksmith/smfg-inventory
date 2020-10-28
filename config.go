package main

import (
	"github.com/rs/zerolog/log"
	sc "github.com/sksmith/go-spring-config"
	"strconv"
	"time"
)

type AppConfig struct {
	Port           string
	GenerateRoutes bool
	LogLevel       string
	LogText        bool
	InMemoryDb     bool
	DbHost         string
	DbPort         string
	DbUser         string
	DbPass         string
	DbName         string
	DbMigrate      bool
	QHost          string
	QPort          string
	QUser          string
	QPass          string
	QName          string
}

const maxRetries = 12
const retryBackoffSec = 5

func LoadConfigs(url, branch, profile string) (*AppConfig, error) {
	appConfig := &AppConfig{}
	var config *sc.Config
	var err error

	for tryCount := 1; tryCount < maxRetries; tryCount++ {
		config, err = sc.Load(url, "smfg-inventory", branch, profile)
		if err == nil {
			break
		}
		log.Error().Err(err).Msg("failed to load configurations... retrying")
		time.Sleep(5 * time.Second)
	}

	if err != nil {
		log.Warn().Err(err).Msg("unable to read configurations from config server")
	} else {
		// API Configs
		appConfig.Port = config.Get("app.port")
		appConfig.GenerateRoutes = getBool(config, "generate.routes")

		// Log Configs
		appConfig.LogLevel = config.Get("log.level")
		appConfig.LogText = getBool(config, "log.text")

		// DB Configs
		appConfig.DbHost = config.Get("db.host")
		appConfig.DbPort = config.Get("db.port")
		appConfig.DbUser = config.Get("db.user")
		appConfig.DbPass = config.Get("db.pass")
		appConfig.DbName = config.Get("db.name")
		appConfig.DbMigrate = getBool(config, "db.migrate")
		appConfig.InMemoryDb = getBool(config, "in.memory")

		// Queue Configs
		appConfig.QHost = config.Get("queue.host")
		appConfig.QPort = config.Get("queue.port")
		appConfig.QUser = config.Get("queue.user")
		appConfig.QPass = config.Get("queue.pass")
		appConfig.QName = config.Get("queue.name")
	}

	return appConfig, nil
}

func getBool(c *sc.Config, property string) bool {
	val, err := strconv.ParseBool(c.Get(property))
	if err != nil {
		return false
	}
	return val
}