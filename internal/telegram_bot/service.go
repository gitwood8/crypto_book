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

	// clear all sessions on startup to ensure clean state
	s.sessions.clearAllSessions()

	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Clean old sessions and log statistics
				beforeCount := s.sessions.getSessionCount()
				s.sessions.cleanOldSessions(5 * time.Minute)

				// Log session statistics every 30 minutes
				if time.Now().Minute()%30 == 0 {
					currentCount := s.sessions.getSessionCount()
					log.Infof("session stats - active: %d, cleaned this cycle: %d",
						currentCount, beforeCount-currentCount)
				}

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

func (s *Service) handleUpdate(ctx context.Context, update tgbotapi.Update) error {
	// panic recovery
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("panic recovered in handleUpdate: %v", r)

			var chatID int64
			var userID int64

			// try to extract chat and user info for recovery message
			if update.CallbackQuery != nil {
				chatID = update.CallbackQuery.Message.Chat.ID
				userID = update.CallbackQuery.From.ID
			} else if update.Message != nil {
				chatID = update.Message.Chat.ID
				userID = update.Message.From.ID
			}

			if chatID != 0 && userID != 0 {
				// Clear any corrupted session
				s.sessions.clearSession(userID)

				// Send recovery message
				recoveryMsg := tgbotapi.NewMessage(chatID,
					"⚠️ Something went wrong. The service was restarted and your session was lost.\nPlease use /start to begin again.")

				if _, err := s.bot.Send(recoveryMsg); err != nil {
					log.Errorf("failed to send recovery message: %v", err)
				}
			}
		}
	}()

	var tgUserID int64

	if update.CallbackQuery != nil {
		tgUserID = update.CallbackQuery.From.ID
	} else if update.Message != nil {
		tgUserID = update.Message.From.ID
	} else {
		return nil
	}

	// log.Info(tgUserID)

	userSession, exists := s.sessions.getSessionVars(tgUserID)
	if !exists && update.CallbackQuery != nil {
		// This is likely a callback from before restart
		return s.handleStaleCallback(update.CallbackQuery)
	}

	switch {
	case update.Message != nil && update.Message.Text == "/start":
		return s.handleStart(ctx, update.Message)

	// case update.Message != nil && update.Message.Text == "jopa":
	// 	mainMenu := tgbotapi.NewMessage(update.Message.Chat.ID, "Welcome back! What would you like to do?")
	// 	mainMenu.ReplyMarkup = tgbotapi.NewReplyKeyboard(
	// 		tgbotapi.NewKeyboardButtonRow(
	// 			tgbotapi.NewKeyboardButton("Portfolios"),
	// 			tgbotapi.NewKeyboardButton("New Portfolio"),
	// 		),
	// 		tgbotapi.NewKeyboardButtonRow(
	// 			tgbotapi.NewKeyboardButton("Help"),
	// 		),
	// 	)
	// 	return s.sendTemporaryMessage(mainMenu, update.Message.From.ID, 20*time.Second)

	case update.Message != nil && update.Message.Text == "qwe":
		resp := tgbotapi.NewMessage(update.Message.Chat.ID, "jopa")
		err := s.sendTgMessage(resp, update.Message.From.ID)
		if err != nil {
			return err
		}
		p, _ := s.sessions.getSessionVars(update.Message.From.ID)

		time.Sleep(3 * time.Second)
		return s.sendTestMessage(update.Message.Chat.ID, p.BotMessageID, "test passed")

	case update.Message != nil && update.Message.Text == "/debug":
		// debug command to show session statistics
		sessionCount := s.sessions.getSessionCount()
		debugMsg := fmt.Sprintf("🔧 Debug Info:\n• Active sessions: %d", sessionCount)

		resp := tgbotapi.NewMessage(update.Message.Chat.ID, debugMsg)
		_, err := s.bot.Send(resp)
		return err

	// catch any callback
	case update.CallbackQuery != nil:
		return s.handleCallback(ctx, update.CallbackQuery, userSession, tgUserID)

	// catch any message
	case update.Message != nil:
		return s.handleMessage(ctx, update.Message, userSession, tgUserID)
	}

	return nil
}

// handle callbacks from before service restart
func (s *Service) handleStaleCallback(cb *tgbotapi.CallbackQuery) error {
	chatID := cb.Message.Chat.ID
	userID := cb.From.ID

	log.Infof("handling stale callback from user %d: %s", userID, cb.Data)

	// handle special case where user clicks the "Start Over" button
	if cb.Data == "/start" {
		// delete the recovery message
		_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(chatID, cb.Message.MessageID))

		mockMsg := &tgbotapi.Message{
			Chat: &tgbotapi.Chat{ID: chatID},
			From: cb.From,
			Text: "/start",
		}

		return s.handleStart(context.Background(), mockMsg)
	}

	// try to delete the old message that user clicked on
	_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(chatID, cb.Message.MessageID))

	recoveryMsg := tgbotapi.NewMessage(chatID,
		"🔄 Service was restarted and your previous session was cleared.\nPlease use /start to begin again.")
	recoveryMsg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🚀 Start Over", "/start"),
		),
	)

	_, err := s.bot.Send(recoveryMsg)
	return err
}
