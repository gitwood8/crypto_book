package telegram_bot

import (
	"context"
	"fmt"
	"time"

	"gitlab.com/avolkov/wood_post/pkg/log"
	"gitlab.com/avolkov/wood_post/store"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Service struct {
	bot      *tgbotapi.BotAPI
	store    *store.Store
	sessions *SessionManager
}

func New(token string, db *store.Store) (*Service, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create Telegram bot: %w", err)
	}

	return &Service{
		bot:      bot,
		store:    db,
		sessions: NewSessionManager(),
	}, nil
}

func (s *Service) Run(ctx context.Context) error {
	log.Infof("authorized on account %s", s.bot.Self.UserName)

	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.sessions.cleanOldSessions(5 * time.Minute)
			case <-ctx.Done():
				log.Info("session cleaner stopping")
				return
			}
		}
	}()

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := s.bot.GetUpdatesChan(u)

	// listening = long polling
	for {
		select {
		case <-ctx.Done():
			log.Info("bot polling stopped by context")
			return nil

		case update, ok := <-updates:
			if !ok {
				log.Warn("updates channel closed")
				return nil
			}

			if err := s.handleUpdate(ctx, update); err != nil {
				log.Error("update handling error:", err)
			}
		}
	}
}
