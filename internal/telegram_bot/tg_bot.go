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
				10*time.Second)
			if sendErr != nil {
				return fmt.Errorf("failed to notify user about user creation error: %w", err)
			}
			return fmt.Errorf("failed to create user in DB: %w", err)
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
	return s.sendTemporaryMessage(resp, 10*time.Second)
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
	return s.sendTemporaryMessage(msg, 10*time.Second)
}

func (s *Service) checkBeforeCreatePortfolio(ctx context.Context, chatID, tgUserID, dbUserID int64) error {
	limitReached, err := s.store.ReachedPortfolioLimit(ctx, dbUserID)
	if err != nil {
		log.Errorf("could not check portfolios amount: %s", err)
		return s.sendTemporaryMessage(
			tgbotapi.NewMessage(
				chatID,
				"Oh, we could not create portfolio for you, please try again."),
			10*time.Second,
		)
	}
	log.Infof("portfolios limit reached: %t", limitReached)

	if limitReached {
		return s.sendTemporaryMessage(
			tgbotapi.NewMessage(chatID,
				"Sorry, you can create up to 2 portfolios. Gimmi ur munney to create more portfolios oi."),
			10*time.Second,
		)
	}

	s.sessions.setState(tgUserID, "waiting_portfolio_name")
	// return s.sendTgMessage(msg)
	return s.sendTemporaryMessage(tgbotapi.NewMessage(chatID,
		"Please enter the name of your portfolio without special characters:"),
		10*time.Second)
}

func (s *Service) ShowPortfolios(ctx context.Context, chatID, tgUserID, dbUserID int64) error {
	ps, err := s.store.GetPortfolios(ctx, dbUserID)
	if err != nil {
		log.Errorf("could not show portfolios: %s", err)
		return s.sendTemporaryMessage(
			tgbotapi.NewMessage(chatID,
				"Sorry, we can get your portfolios, please try again later."),
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
		return s.sendTemporaryMessage(msg, 10*time.Second)
	}

	log.Infof("user_id: %d, portfolios: %s", dbUserID, ps)

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, p := range ps {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(p, "portfolio_"+p),
		))
	}

	msg := tgbotapi.NewMessage(chatID, "Select a portfolio to perform an action:")
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)

	return s.sendTemporaryMessage(msg, 10*time.Second)
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
