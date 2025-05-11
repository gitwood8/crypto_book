package telegram_bot

import (
	"context"
	"fmt"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
	"gitlab.com/avolkov/wood_post/pkg/log"
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
		// fmt.Println("eqweqweqw")
		return s.handleMessage(ctx, update.Message)
	}

	return nil
}

func (s *Service) handleStart(ctx context.Context, msg *tgbotapi.Message) error {
	tgUserID := msg.From.ID

	exists, err := s.store.UserExists(ctx, tgUserID)
	if err != nil {
		return errors.Wrap(err, "failed to check user existence")
	}

	if !exists {
		if err := s.store.CreateUserIfNotExists(ctx, tgUserID, msg.From.UserName); err != nil {
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
	tgUserID := cb.From.ID

	switch cb.Data {
	case "create_portfolio":
		log.Infof("tg_user_id: %d, selected %s", tgUserID, cb.Data)

		dbUserID, err := s.store.GetUserIDByTelegramID(ctx, tgUserID)
		if err != nil {
			return errors.Wrap(err, "failed to get user from DB")
		}

		limitReached, err := s.store.ReachedPortfolioLimit(ctx, dbUserID)
		if err != nil {
			log.Errorf("could not check portfolios amount: %s", err)
			return s.sendTemporaryMessage(
				tgbotapi.NewMessage(cb.Message.Chat.ID, "Oh, we could not create portfolio for you, please try again."), 10*time.Second,
			)
		}
		log.Infof("portfolios limit reached: %t", limitReached)

		if limitReached {
			return s.sendTemporaryMessage(
				tgbotapi.NewMessage(cb.Message.Chat.ID, "Sorry, you can create up to 2 portfolios. Gimmi ur munney to create more portfolios oi."), 10*time.Second,
			)
		}

		s.sessions.setState(tgUserID, "waiting_portfolio_name")
		// return s.sendTgMessage(msg)
		return s.sendTemporaryMessage(tgbotapi.NewMessage(cb.Message.Chat.ID, "Please enter the name of your portfolio:"), 10*time.Second)
	case "who_am_i":
		// TODO: i dont need set state here, its finidhed flow
		s.sessions.setState(tgUserID, "who_am_i")
		msg := tgbotapi.NewMessage(cb.Message.Chat.ID, "Im very cool bot")
		return s.sendTemporaryMessage(msg, 10*time.Second)
	}

	return nil
}

func (s *Service) handleMessage(ctx context.Context, msg *tgbotapi.Message) error {
	tgUserID := msg.From.ID
	state, ok := s.sessions.getState(tgUserID)

	if !ok {
		return nil
	}

	dbUserID, err := s.store.GetUserIDByTelegramID(ctx, msg.From.ID)
	if err != nil {
		return errors.Wrap(err, "failed to get user from DB")
	}

	switch state {
	case "waiting_portfolio_name":
		nameTaken, err := s.store.PortfolioNameExists(ctx, dbUserID, msg.Text)
		if err != nil {
			log.Errorf("could not check PortfolioNameExists: %s", err)
			return s.sendTemporaryMessage(
				tgbotapi.NewMessage(msg.Chat.ID, "Oh, we could not create portfolio for you, please try again."), 10*time.Second,
			)
		}

		if nameTaken {
			return s.sendTemporaryMessage(
				tgbotapi.NewMessage(msg.Chat.ID, "Portfolio with such name already exists, try another name."), 10*time.Second,
			)
		}

		s.sessions.setTempName(tgUserID, msg.Text)
		s.sessions.setState(tgUserID, "waiting_portfolio_description")

		t := fmt.Sprintf("Please enter description for portfolio %s", msg.Text)

		return s.sendTemporaryMessage(tgbotapi.NewMessage(msg.Chat.ID, t), 10*time.Second)

	case "waiting_portfolio_description":
		portfolioName, _ := s.sessions.getTempName(tgUserID)
		portfolioDesc := msg.Text

		err = s.store.CreatePortfolio(ctx, dbUserID, portfolioName, portfolioDesc)
		if err != nil {
			// switch {
			// case errors.Is(err, store.ErrPortfolioNameExists):
			// 	return s.sendTemporaryMessage(
			// 		tgbotapi.NewMessage(msg.Chat.ID, "You already have a portfolio with this name. Try again."), 10*time.Second,
			// 	)
			// case errors.Is(err, store.ErrPortfolioLimitReached):
			// 	return s.sendTemporaryMessage(
			// 		tgbotapi.NewMessage(msg.Chat.ID, "Sorry, you can create up to 2 portfolios. Gimmi ur munney to create more portfolios oi."), 10*time.Second,
			// 	)
			// default:
			// 	return fmt.Errorf("failed to create portfolio: %w", err)
			// }
			return fmt.Errorf("failed to create portfolio: %w", err)
		}

		s.sessions.clearSession(tgUserID)

		return s.sendTemporaryMessage(tgbotapi.NewMessage(msg.Chat.ID, "Portfolio created successfully."), 15*time.Second)

	// TODO: investigate it (delete)
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
