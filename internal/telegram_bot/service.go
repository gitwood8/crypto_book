package telegram_bot

import (
	"context"
	"fmt"
	"time"

	"gitlab.com/avolkov/wood_post/config"
	"gitlab.com/avolkov/wood_post/pkg/log"
	"gitlab.com/avolkov/wood_post/store"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Service struct {
	bot      *tgbotapi.BotAPI
	store    *store.Store
	sessions *SessionManager
	cfg      *config.Config
}

func New(token string, db *store.Store, cfg *config.Config) (*Service, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create Telegram bot: %w", err)
	}

	return &Service{
		bot:      bot,
		store:    db,
		sessions: NewSessionManager(),
		cfg:      cfg,
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

func (s *Service) handleUpdate(ctx context.Context, update tgbotapi.Update) error {
	// add panic recovery to prevent service crashes
	defer func() {
		if r := recover(); r != nil {
			log.Error("panic recovered in handleUpdate", "panic", r, "stack", fmt.Sprintf("%+v", r))

			// try to send recovery message if we can identify the user
			var chatID int64
			var userID int64

			if update.CallbackQuery != nil {
				chatID = update.CallbackQuery.Message.Chat.ID
				userID = update.CallbackQuery.From.ID
			} else if update.Message != nil {
				chatID = update.Message.Chat.ID
				userID = update.Message.From.ID
			}

			if chatID != 0 && userID != 0 {
				// clear any existing session for this user
				s.sessions.clearSession(userID)

				// Delete the problematic message if it's a callback
				if update.CallbackQuery != nil && update.CallbackQuery.Message != nil {
					_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(chatID, update.CallbackQuery.Message.MessageID))
				}

				// send recovery message
				recoveryMsg := tgbotapi.NewMessage(chatID,
					"🔧 *Service Recovery*\n\n"+
						"The service was recently restarted. Your previous session has been cleared.\n\n"+
						"Please start fresh by using /start or the main menu.")
				recoveryMsg.ParseMode = "Markdown"
				recoveryMsg.ReplyMarkup = s.showMainMenu(chatID, userID)

				if _, err := s.bot.Send(recoveryMsg); err != nil {
					log.Error("failed to send recovery message", "error", err)
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

	// get or create session - this ensures session exists
	userSession, sessionExists := s.sessions.getSessionVars(tgUserID)

	// if this is a callback query but no session exists, it means the service was restarted
	// and the user is clicking on an old button
	if update.CallbackQuery != nil && !sessionExists {
		chatID := update.CallbackQuery.Message.Chat.ID
		messageID := update.CallbackQuery.Message.MessageID

		// delete the old message
		_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(chatID, messageID))

		// send session expired message
		expiredMsg := tgbotapi.NewMessage(chatID,
			"⚠️ *Session Expired*\n\n"+
				"This button is from before the service restart. Please use the main menu below or enter /start.")
		expiredMsg.ParseMode = "Markdown"

		err := s.sendTemporaryMessage(expiredMsg, tgUserID, 5*time.Second)
		if err != nil {
			return err
		}

		return s.showMainMenu(chatID, tgUserID)
	}

	switch {
	case update.Message != nil && update.Message.Text == "/start":
		return s.handleStart(ctx, update.Message)

	// case update.Message != nil && update.Message.Text == "qwe":
	// 	resp := tgbotapi.NewMessage(update.Message.Chat.ID, "jopa")
	// 	err := s.sendTgMessage(resp, update.Message.From.ID)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	p, _ := s.sessions.getSessionVars(update.Message.From.ID)

	// 	time.Sleep(3 * time.Second)
	// 	return s.sendTestMessage(update.Message.Chat.ID, p.BotMessageID, "test passed")

	// catch any callback
	case update.CallbackQuery != nil:
		return s.handleCallback(ctx, update.CallbackQuery, userSession, tgUserID)

	// catch any message
	case update.Message != nil:
		return s.handleMessage(ctx, update.Message, userSession, tgUserID)
	}

	return nil
}
