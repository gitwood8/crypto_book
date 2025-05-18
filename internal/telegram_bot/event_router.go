package telegram_bot

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
	"gitlab.com/avolkov/wood_post/pkg/log"
)

func (s *Service) handleUpdate(ctx context.Context, update tgbotapi.Update) error {
	// log.Info(update.CallbackQuery)
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

func (s *Service) handleCallback(ctx context.Context, cb *tgbotapi.CallbackQuery) error {
	tgUserID := cb.From.ID
	// TODO: check this log

	dbUserID, err := s.store.GetUserIDByTelegramID(ctx, tgUserID)
	if err != nil {
		return errors.Wrap(err, "failed to get user from DB")
	}

	log.Infof("user_id: %d, selected callback: %s", dbUserID, cb.Data)

	switch /* cb.Data */ {
	case cb.Data == "create_portfolio":
		return s.checkBeforeCreatePortfolio(ctx, cb.Message.Chat.ID, tgUserID, dbUserID)
	// case "who_am_i":
	// TODO: will be added later

	// s.sessions.setState(tgUserID, "who_am_i")
	// msg := tgbotapi.NewMessage(cb.Message.Chat.ID, "Im very cool bot")
	// return s.sendTemporaryMessage(msg, 10*time.Second)

	case cb.Data == "show_portfolios":
		return s.ShowPortfolios(ctx, cb.Message.Chat.ID, tgUserID, dbUserID)

	case strings.HasPrefix(cb.Data, "portfolio_"):
		return s.ShowPortfolioActions(ctx, cb.Data, cb.Message.Chat.ID, tgUserID)

	case cb.Data == "get_report_from_portfolio":
		return nil
	case cb.Data == "set_portfolio_as_default":
		return nil
	case cb.Data == "rename_portfolio":
		return nil
	case cb.Data == "delete_portfolio":
		return nil
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
		deleteMsg := tgbotapi.NewDeleteMessage(msg.Chat.ID, msg.MessageID)
		_, _ = s.bot.Request(deleteMsg)

		r := regexp.MustCompile(`[^a-zA-Z0-9_]+`)
		pName := r.ReplaceAllString(strings.ReplaceAll(msg.Text, " ", "_"), "")

		ru := regexp.MustCompile(`_+`)
		pName = ru.ReplaceAllString(pName, "_")

		nameTaken, err := s.store.PortfolioNameExists(ctx, dbUserID, pName)
		if err != nil {
			log.Errorf("could not check PortfolioNameExists: %s", err)
			return s.sendTemporaryMessage(
				tgbotapi.NewMessage(msg.Chat.ID,
					"Oh, we could not create portfolio for you, please try again."),
				10*time.Second,
			)
		}

		if nameTaken {
			return s.sendTemporaryMessage(
				tgbotapi.NewMessage(msg.Chat.ID,
					"Portfolio with such name already exists, try another name."),
				10*time.Second,
			)
		}

		s.sessions.setTempField(tgUserID, "TempPortfolioName", pName)
		s.sessions.setState(tgUserID, "waiting_portfolio_description")

		t := fmt.Sprintf("Please enter description for portfolio: *%s*", pName)

		return s.sendTemporaryMessage(tgbotapi.NewMessage(msg.Chat.ID, t), 10*time.Second)

	case "waiting_portfolio_description":
		deleteMsg := tgbotapi.NewDeleteMessage(msg.Chat.ID, msg.MessageID)
		_, _ = s.bot.Request(deleteMsg)

		portfolio, _ := s.sessions.getSessionVars(tgUserID)
		portfolioDesc := msg.Text

		err = s.store.CreatePortfolio(ctx, dbUserID, portfolio.TempPortfolioName, portfolioDesc)
		if err != nil {
			return fmt.Errorf("failed to create portfolio: %w", err)
		}

		s.sessions.clearSession(tgUserID)

		return s.sendTemporaryMessage(tgbotapi.NewMessage(msg.Chat.ID,
			"Portfolio created successfully."),
			10*time.Second)

	// TODO: investigate it (delete)
	case "who_am_i":
		return nil
	}

	return nil
}
