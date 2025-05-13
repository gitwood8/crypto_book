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

func (s *Service) Run() error {
	log.Infof("authorized on account %s", s.bot.Self.UserName)

	ctx := context.Background()

	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		// log.Info("clearing sessions")
		defer ticker.Stop()

		for range ticker.C {
			// log.Info("cleaning")
			s.sessions.cleanOldSessions(5 * time.Minute)
		}
	}()

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := s.bot.GetUpdatesChan(u)

	// listening = long polling
	for update := range updates {
		if err := s.handleUpdate(ctx, update); err != nil {
			log.Error("update handling error:", err)
		}
	}

	return nil
}
