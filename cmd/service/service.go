package main

import (
	"context"
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		log.Info(ctx)
		err := services.TelegramBot.Run(ctx)
		if err != nil {
			log.Error("bot error:", err)
		}
	}()

	log.Info("main: bot started polling")

	// Graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	log.Info("Shutting down...")
	cancel()
}
