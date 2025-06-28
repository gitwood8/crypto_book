package telegram_bot

import (
	"context"
	"fmt"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
	t "gitlab.com/avolkov/wood_post/pkg/types"
)

func (s *Service) handleStart(ctx context.Context, msg *tgbotapi.Message) error {
	tgUserID := msg.From.ID

	exists, err := s.store.UserExists(ctx, tgUserID)
	if err != nil {
		return errors.Wrap(err, "failed to check user existence")
	}

	if !exists {
		err := s.store.CreateUserIfNotExists(ctx, tgUserID, msg.From.UserName)
		if err != nil {
			sendErr := s.sendTemporaryMessage(
				tgbotapi.NewMessage(msg.Chat.ID,
					"Failed to create user. Please try again later."),
				tgUserID,
				20*time.Second)

			if sendErr != nil {
				return fmt.Errorf("failed to notify user about user creation error: %w", err)
			}
			return fmt.Errorf("failed to create user in DB: %w", err)
		}

		return s.showWelcome(msg.Chat.ID, tgUserID)
	}

	return s.showMainMenu(msg.Chat.ID, tgUserID)
}

func (s *Service) showWelcome(chatID, tgUserID int64) error {
	msg := tgbotapi.NewMessage(chatID, "Welcome! Let's create your first portfolio.")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Create portfolio", "create_portfolio"),
			tgbotapi.NewInlineKeyboardButtonData("Who am I?", "who_am_i"),
		),
	)
	// return s.sendTgMessage(msg, tgUserID)
	return s.sendTemporaryMessage(msg, tgUserID, 20*time.Second)
}

func (s *Service) showMainMenu(chatID, tgUserID int64) error {
	s.sessions.setState(tgUserID, "main_menu")

	mainMenu := tgbotapi.NewMessage(chatID, "What would you like to do next?")
	mainMenu.ReplyMarkup = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("My portfolios"),
			tgbotapi.NewKeyboardButton("Transactions"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Help"), // not valid yet
		),
	)

	return s.sendTemporaryMessage(mainMenu, tgUserID, 20*time.Second)
}

func (s *Service) showServiceInfo(chatID, tgUserID int64) error {
	msg := tgbotapi.NewMessage(chatID, t.ServiceDescription)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Back", "cancel_action"),
		),
	)

	return s.sendTemporaryMessage(msg, tgUserID, 200*time.Second)
}

func (s *Service) sendTgMessage(msg tgbotapi.Chattable, tgUserID int64) error {
	sentMsg, err := s.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	// fmt.Println("bot message id from func 1: ", sentMsg.MessageID)
	s.sessions.setTempField(tgUserID, "BotMessageID", sentMsg.MessageID)
	return nil
}

func (s *Service) sendTemporaryMessage(msg tgbotapi.Chattable, tgUserID int64, delay time.Duration) error {
	sentMsg, err := s.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("failed to send temporary message: %w:", err)
	}

	s.sessions.setTempField(tgUserID, "BotMessageID", sentMsg.MessageID)

	go func() {
		time.Sleep(delay)
		deleteMsg := tgbotapi.NewDeleteMessage(sentMsg.Chat.ID, sentMsg.MessageID)
		_, _ = s.bot.Request(deleteMsg)
		// s.sessions.clearSession(tgUserID) // added 1st June
	}()

	return nil
}

func (s *Service) editMessageText(chatID int64, messageID int, text string) error {
	// fmt.Println("editMessageText started")
	// fmt.Println(chatID, messageID, text)
	edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
	// edit.ParseMode = "Markdown"
	_, err := s.bot.Send(edit)
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) sendTestMessage(chatID int64, messageID int, text string) error {
	fmt.Println("editMessageText started: ", chatID, messageID, text)
	edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
	edit.ParseMode = "Markdown"
	_, err := s.bot.Send(edit)

	if err != nil {
		return err
	}
	return nil
}
