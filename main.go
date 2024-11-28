package main

import (
	"os"

	"github.com/bjarneo/pipe/internal/config"
	"github.com/bjarneo/pipe/internal/deploy"
	"github.com/bjarneo/pipe/internal/logger"
)

func main() {
	log := initLogger()
	defer log.Close()

	cfg := config.Load()

	if cfg.Rollback {
		if err := deploy.Rollback(&cfg, log); err != nil {
			log.Error("Rollback failed", err)
			os.Exit(1)
		}
	} else {
		if err := deploy.Deploy(&cfg, log); err != nil {
			log.Error("Deployment failed", err)
			os.Exit(1)
		}
	}
}

func initLogger() *logger.Logger {
	log, err := logger.New("deploy.log")
	if err != nil {
		log.Fatal(err)
	}
	return log
}
