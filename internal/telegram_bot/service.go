package telegram_bot

import (
	"context"
	"fmt"
	"time"

	"gitlab.com/avolkov/wood_post/pkg/log"
	"gitlab.com/avolkov/wood_post/store"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
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

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := s.bot.GetUpdatesChan(u)

	for update := range updates {
		if err := s.handleUpdate(ctx, update); err != nil {
			log.Error("update handling error:", err)
		}
	}

	return nil
}

func (s *Service) handleUpdate(ctx context.Context, update tgbotapi.Update) error {
	switch {
	case update.Message != nil && update.Message.Text == "/start":
		return s.showWelcome(update.Message.Chat.ID)

	case update.CallbackQuery != nil:
		return s.handleCallback(ctx, update.CallbackQuery)

	case update.Message != nil:
		return s.handleMessage(ctx, update.Message)
	}

	return nil
}

func (s *Service) showWelcome(chatID int64) error {
	msg := tgbotapi.NewMessage(chatID, "Welcome! Please create your first portfolio.")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Create portfolio", "create_portfolio"),
		),
	)

	return s.send(msg)
}

// asd
func (s *Service) handleCallback(ctx context.Context, cb *tgbotapi.CallbackQuery) error {
	userID := cb.From.ID

	switch cb.Data {
	case "create_portfolio":
		if err := s.store.CreateUserIfNotExists(ctx, userID, cb.From.UserName); err != nil {
			sendErr := s.sendTemporaryMessage(cb.Message.Chat.ID, "Failed to create user. Please try again later.", 10*time.Second)
			if sendErr != nil {
				return errors.Wrap(sendErr, "failed to notify user about user creation error")
			}
			return errors.Wrap(err, "failed to create user in DB")
		}

		s.sessions.setState(userID, "waiting_portfolio_name")

		msg := tgbotapi.NewMessage(cb.Message.Chat.ID, "Please enter the name of your portfolio:")
		return s.send(msg)
	}

	return nil
}

func (s *Service) handleMessage(ctx context.Context, msg *tgbotapi.Message) error {
	userID := msg.From.ID
	state, ok := s.sessions.getState(userID)

	if !ok {
		return nil
	}

	switch state {
	case "waiting_portfolio_name":
		s.sessions.setState(userID, "waiting_portfolio_description")
		s.sessions.setTempName(userID, msg.Text)

		return s.send(tgbotapi.NewMessage(msg.Chat.ID, "Please enter a description:"))

	case "waiting_portfolio_description":
		name, _ := s.sessions.getTempName(userID)
		description := msg.Text

		dbUserID, err := s.store.GetUserIDByTelegramID(ctx, msg.From.ID)
		if err != nil {
			return errors.Wrap(err, "failed to get user from DB")
		}

		if err := s.store.CreatePortfolio(ctx, dbUserID, name, description); err != nil {
			return errors.Wrap(err, "failed to create portfolio")
		}

		s.sessions.clearSession(userID)

		return s.send(tgbotapi.NewMessage(msg.Chat.ID, "Portfolio created successfully."))
	}

	return nil
}

// Универсальная отправка сообщений
func (s *Service) send(c tgbotapi.Chattable) error {
	if _, err := s.bot.Send(c); err != nil {
		return errors.Wrap(err, "failed to send telegram message")
	}
	return nil
}

func (s *Service) sendTemporaryMessage(chatID int64, text string, lifetime time.Duration) error {
	msg := tgbotapi.NewMessage(chatID, text)

	sentMsg, err := s.bot.Send(msg)
	if err != nil {
		return errors.Wrap(err, "failed to send temporary message")
	}

	go func() {
		time.Sleep(lifetime)

		if err := s.deleteMessage(sentMsg.Chat.ID, sentMsg.MessageID); err != nil {
			log.Warnf("failed to delete temporary message: %v", err)
		}
	}()

	return nil
}

func (s *Service) deleteMessage(chatID int64, messageID int) error {
	req := tgbotapi.DeleteMessageConfig{
		ChatID:    chatID,
		MessageID: messageID,
	}

	if _, err := s.bot.Request(req); err != nil {
		return errors.Wrap(err, "failed to delete telegram message")
	}

	return nil
}
