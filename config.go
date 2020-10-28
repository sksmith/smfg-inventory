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
	QHost          string
	QPort          string
	QUser          string
	QPass          string
	QName          string
}

func LoadConfigs(url, branch, profile string) (*AppConfig, error) {
	appConfig := &AppConfig{}
	var config *sc.Config
	var err error
	const maxRetries = 5
	tryCount := 0

	for {
		tryCount++
		if tryCount > maxRetries {
			break
		}
		config, err = sc.Load(url, "smfg-inventory", branch, profile)
		if err == nil {
			break
		}
		log.Error().Err(err).Msg("failed to load configurations... retrying")
		time.Sleep(2 * time.Second)
	}


	if err != nil {
		log.Warn().Err(err).Msg("unable to read configurations from config server")
	} else {
		appConfig.Port = config.Get("app.port")
		appConfig.LogLevel = config.Get("log.level")
		appConfig.DbHost = config.Get("db.host")
		appConfig.DbPort = config.Get("db.port")
		appConfig.DbUser = config.Get("db.user")
		appConfig.DbPass = config.Get("db.pass")
		appConfig.DbName = config.Get("db.name")
		appConfig.QHost = config.Get("queue.host")
		appConfig.QPort = config.Get("queue.port")
		appConfig.QUser = config.Get("queue.user")
		appConfig.QPass = config.Get("queue.pass")
		appConfig.QName = config.Get("queue.name")

		routes, err := strconv.ParseBool(config.Get("generate.routes"))
		if err != nil {
			routes = false
		}
		appConfig.GenerateRoutes = routes

		text, err := strconv.ParseBool(config.Get("log.text"))
		if err != nil {
			text = false
		}
		appConfig.LogText = text

		memory, err := strconv.ParseBool(config.Get("in.memory"))
		if err != nil {
			memory = false
		}
		appConfig.InMemoryDb = memory
	}

	return appConfig, nil
}
