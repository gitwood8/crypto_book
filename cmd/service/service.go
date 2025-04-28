package main

import (
	"os"
	"os/signal"
	"syscall"

	"gitlab.com/avolkov/wood_post/pkg/log"

	"gitlab.com/avolkov/wood_post/config"
	"gitlab.com/avolkov/wood_post/internal"
)

func main() {
	log.Info("main: starting service")

	cfg := config.Load()
	// log.Info("config loaded") asdasda

	services, err := internal.New(cfg)
	if err != nil {
		log.Error("init error:", err)
		return
	}
	// log.Info("internal.New completed")

	go func() {
		if err := services.TelegramBot.Run(); err != nil {
			log.Error("bot error:", err)
		}
	}()

	log.Info("main: bot started polling")

	// Graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	log.Info("Shutting down...")
}
