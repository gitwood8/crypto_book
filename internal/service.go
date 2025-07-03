package internal

import (
	"gitlab.com/avolkov/wood_post/config"
	"gitlab.com/avolkov/wood_post/internal/telegram_bot"
	"gitlab.com/avolkov/wood_post/pkg/log"
	"gitlab.com/avolkov/wood_post/store"
)

type Services struct {
	TelegramBot *telegram_bot.Service
	// Store       *store.Store
}

func New(cfg *config.Config) (*Services, error) {

	db, err := store.New(cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)
	if err != nil {
		return nil, err
	}
	log.Info("internal: db connection established")

	tg, err := telegram_bot.New(cfg.TelegramBotToken, db, cfg) //FIXME
	if err != nil {
		return nil, err
	}
	log.Info("internal: telegram bot is runnig")

	return &Services{
		TelegramBot: tg,
	}, nil
}
