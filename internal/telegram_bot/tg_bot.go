package telegram_bot

import (
	"context"
	"fmt"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
)

func (s *Service) handleUpdate(ctx context.Context, update tgbotapi.Update) error {
	switch {
	case update.Message != nil && update.Message.Text == "/start":
		return s.handleStart(ctx, update.Message)

	case update.Message != nil && update.Message.Text == "/qwe":
		return s.sendTestMessage(update.Message.From.ID, "test passed")

	// catch any callback
	case update.CallbackQuery != nil:
		return s.handleCallback(ctx, update.CallbackQuery)

	// catch any message
	case update.Message != nil:
		fmt.Println("eqweqweqw")
		return s.handleMessage(ctx, update.Message)
	}

	return nil
}

func (s *Service) handleStart(ctx context.Context, msg *tgbotapi.Message) error {
	userID := msg.From.ID

	exists, err := s.store.UserExists(ctx, userID)
	if err != nil {
		return errors.Wrap(err, "failed to check user existence")
	}

	if !exists {
		if err := s.store.CreateUserIfNotExists(ctx, userID, msg.From.UserName); err != nil {
			notify := tgbotapi.NewMessage(msg.Chat.ID, "Failed to create user. Please try again later.")
			if sendErr := s.sendTemporaryMessage(notify, 10*time.Second); sendErr != nil {
				return errors.Wrap(sendErr, "failed to notify user about user creation error")
			}
			return errors.Wrap(err, "failed to create user in DB")
		}

		return s.showWelcome(msg.Chat.ID)
	}

	resp := tgbotapi.NewMessage(msg.Chat.ID, "You already have an account. What would you like to do next?")
	resp.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("My portfolios", "show_portfolios"),
			tgbotapi.NewInlineKeyboardButtonData("New portfolio", "create_portfolio"),
		),
	)
	return s.sendTemporaryMessage(resp, 60*time.Second)
}

func (s *Service) showWelcome(chatID int64) error {
	msg := tgbotapi.NewMessage(chatID, "Welcome! Let's create your first portfolio.")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Create portfolio", "create_portfolio"),
			tgbotapi.NewInlineKeyboardButtonData("Who am I?", "who_am_i"),
		),
	)

	// return s.sendTgMessage(msg)
	return s.sendTemporaryMessage(msg, 60*time.Second)
}

func (s *Service) handleCallback(ctx context.Context, cb *tgbotapi.CallbackQuery) error {
	userID := cb.From.ID

	switch cb.Data {
	case "create_portfolio":
		s.sessions.setState(userID, "waiting_portfolio_name")
		msg := tgbotapi.NewMessage(cb.Message.Chat.ID, "Please enter the name of your portfolio:")
		// return s.sendTgMessage(msg)
		return s.sendTemporaryMessage(msg, 60*time.Second)
	case "who_am_i":
		s.sessions.setState(userID, "who_am_i")
		msg := tgbotapi.NewMessage(cb.Message.Chat.ID, "Im very cool bot")
		return s.sendTemporaryMessage(msg, 60*time.Second)
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

		return s.sendTgMessage(tgbotapi.NewMessage(msg.Chat.ID, "Please enter a description:"))

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

		return s.sendTgMessage(tgbotapi.NewMessage(msg.Chat.ID, "Portfolio created successfully."))

	// TODO: investigate it
	case "who_am_i":
		return nil
	}

	return nil
}

func (s *Service) sendTgMessage(c tgbotapi.Chattable) error {
	if _, err := s.bot.Send(c); err != nil {
		return errors.Wrap(err, "failed to send telegram message")
	}
	return nil
}

func (s *Service) sendTemporaryMessage(msg tgbotapi.Chattable, delay time.Duration) error {
	sentMsg, err := s.bot.Send(msg)
	if err != nil {
		return errors.Wrap(err, "failed to send temporary message")
	}

	go func() {
		time.Sleep(delay)
		deleteMsg := tgbotapi.NewDeleteMessage(sentMsg.Chat.ID, sentMsg.MessageID)
		_, _ = s.bot.Request(deleteMsg)
	}()

	return nil
}

func (s *Service) sendTestMessage(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	_, err := s.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("failed to send text message: %w", err)
	}
	return nil
}
