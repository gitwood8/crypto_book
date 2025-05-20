package telegram_bot

import (
	"context"
	"fmt"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
	"gitlab.com/avolkov/wood_post/pkg/log"
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
				10*time.Second)

			if sendErr != nil {
				return fmt.Errorf("failed to notify user about user creation error: %w", err)
			}
			return fmt.Errorf("failed to create user in DB: %w", err)
		}

		return s.showWelcome(msg.Chat.ID, tgUserID)
	}

	// here should be buttons with general flow buttons (transaction, portfolio and reports)
	resp := tgbotapi.NewMessage(msg.Chat.ID, "You already have an account. What would you like to do next?")
	resp.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("My portfolios", "show_portfolios"),
			tgbotapi.NewInlineKeyboardButtonData("New portfolio", "create_portfolio"),
		),
	)

	// return s.sendTgMessage(resp, tgUserID)
	return s.sendTemporaryMessage(resp, tgUserID, 10*time.Second)
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

func (s *Service) checkBeforeCreatePortfolio(ctx context.Context, chatID, tgUserID, dbUserID int64) error {
	r, ok := s.sessions.getSessionVars(tgUserID)
	if !ok {
		return nil
	}

	limitReached, err := s.store.ReachedPortfolioLimit(ctx, dbUserID)
	if err != nil {
		log.Errorf("could not check portfolios amount: %s", err)
		return s.sendTemporaryMessage(
			tgbotapi.NewMessage(
				chatID,
				"Oh, we could not create portfolio for you, please try again."),
			tgUserID,
			10*time.Second,
		)
	}
	log.Infof("user_id: %d, portfolios limit reached: %t", dbUserID, limitReached)

	if limitReached {
		return s.editMessageText(
			chatID,
			r.BotMessageID,
			"Sorry, you can create up to 2 portfolios. Gimmi ur munney to create more portfolios oi.")
	}

	s.sessions.setState(tgUserID, "waiting_portfolio_name")

	return s.editMessageText(
		chatID,
		r.BotMessageID,
		"Please enter the name of your portfolio without special characters:")
}

func (s *Service) ShowPortfolioActions(ctx context.Context, cb string, chatID, tgUserID int64) error {
	s.sessions.setTempField(tgUserID, "SelectedPortfolioName", cb)
	fmt.Printf("chosen portfolio: %s", cb)

	type Action struct {
		TgText       string
		CallBackName string
	}

	actions := []Action{
		{"Get report", "get_report_from_portfolio"},
		{"Set as default", "set_portfolio_as_default"},
		{"Rename", "rename_portfolio"},
		{"Delete", "delete_portfolio"},
		{"Create new", "create_portfolio"},
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, a := range actions {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(a.TgText, a.CallBackName),
		))
	}

	msg := tgbotapi.NewMessage(chatID, "What would you like to do with portfolio?")
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)

	return s.sendTemporaryMessage(msg, tgUserID, 10*time.Second)
}

func (s *Service) ShowPortfolios(ctx context.Context, chatID, tgUserID, dbUserID int64) error {
	r, _ := s.sessions.getSessionVars(tgUserID)
	deleteMsg := tgbotapi.NewDeleteMessage(chatID, r.BotMessageID)
	_, _ = s.bot.Request(deleteMsg)

	ps, err := s.store.GetPortfolios(ctx, dbUserID)
	if err != nil {
		log.Errorf("could not show portfolios: %s", err)
		return s.sendTemporaryMessage(
			tgbotapi.NewMessage(chatID,
				"Sorry, we can get your portfolios, please try again later."),
			tgUserID,
			10*time.Second,
		)
	}

	if len(ps) == 0 {
		msg := tgbotapi.NewMessage(chatID, "You have no portfolios, let's create one!")
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("New portfolio", "create_portfolio"),
			),
		)
		return s.sendTemporaryMessage(msg, tgUserID, 10*time.Second)
	}

	log.Infof("user_id: %d, portfolios list: %s", dbUserID, ps)

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, p := range ps {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(p, "portfolio_"+p),
		))
	}

	msg := tgbotapi.NewMessage(chatID, "Select a portfolio to perform an action:")
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)

	return s.sendTemporaryMessage(msg, tgUserID, 10*time.Second)
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

// func (s *Service) sendTestMessage(chatID int64, text string) error {
// 	msg := tgbotapi.NewMessage(chatID, text)
// 	_, err := s.bot.Send(msg)
// 	if err != nil {
// 		return fmt.Errorf("failed to send text message: %w", err)
// 	}
// 	return nil
// }

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
