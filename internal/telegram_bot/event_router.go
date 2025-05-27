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

	case update.Message != nil && update.Message.Text == "qwe":
		// fmt.Println(update.Message.Text)
		resp := tgbotapi.NewMessage(update.Message.Chat.ID, "jopa")
		err := s.sendTgMessage(resp, update.Message.From.ID)
		if err != nil {
			return err
		}
		p, _ := s.sessions.getSessionVars(update.Message.From.ID)

		time.Sleep(3 * time.Second)
		return s.sendTestMessage(update.Message.Chat.ID, p.BotMessageID, "test passed")

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

	r, _ := s.sessions.getSessionVars(tgUserID)
	log.Infof("user_id: %d, selected callback: %s", dbUserID, cb.Data)

	switch /* cb.Data */ {
	case cb.Data == "create_portfolio":
		return s.checkBeforeCreatePortfolio(ctx, cb.Message.Chat.ID, tgUserID, dbUserID)
	// case "who_am_i":
	// TODO: will be added later

	// s.sessions.setState(tgUserID, "who_am_i") // why?
	// msg := tgbotapi.NewMessage(cb.Message.Chat.ID, "Im very cool bot")
	// return s.sendTemporaryMessage(msg, 10*time.Second)

	// case cb.Data == "show_portfolios":
	// 	return s.ShowPortfolios(ctx, cb.Message.Chat.ID, tgUserID, dbUserID, r.BotMessageID)

	// case strings.HasPrefix(cb.Data, "portfolio_"):
	// 	// log.Infof("callback received: %+v", cb)
	// 	return s.ShowPortfolioActions(cb.Data, cb.Message.Chat.ID, tgUserID, r.BotMessageID)

	// case cb.Data == "get_report_from_portfolio":
	// 	return nil
	// case cb.Data == "set_portfolio_as_default":
	// 	return nil
	// case cb.Data == "rename_portfolio":
	// 	return nil

	// case cb.Data == "delete_portfolio": TESTING

	case cb.Data == "gf_portfolios":
		return s.gfPortfoliosMain(cb.Message.Chat.ID, tgUserID, r.BotMessageID)

	case cb.Data == "gf_portfolios_delete":
		s.sessions.setTempField(tgUserID, "NextAction", "delete")
		return s.ShowPortfolios(ctx, cb.Message.Chat.ID, tgUserID, dbUserID, r.BotMessageID, "delete")

		// TODO тут должна быть функция, которая разбирает кейс Action (delete/rename/set default ...)
	case strings.Contains(cb.Data, "::"):
		log.Infof("callback data: %s", cb.Data)

		// return s.askDeletePortfolioConfirmation(cb.Message.Chat.ID, tgUserID, r.BotMessageID)
		return s.performActionForPortfolio(ctx, cb.Message.Chat.ID, tgUserID, dbUserID, r.BotMessageID, cb.Data)

	case cb.Data == "confirm_portfolio_deletinon":
		return s.portfolioDeletinonConfirmed(ctx, cb.Message.Chat.ID, tgUserID, dbUserID, r.BotMessageID, r.SelectedPortfolioName)

	case cb.Data == "cancel_action":
		s.sessions.clearSession(tgUserID)
		// return s.editMessageText(cb.Message.Chat.ID, r.BotMessageID, "")

		deleteMsg := tgbotapi.NewDeleteMessage(cb.Message.Chat.ID, r.BotMessageID)
		_, _ = s.bot.Request(deleteMsg)
		return nil
	}
	// fmt.Println("portfolio, callback: ", action, p)

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
		p, _ := s.sessions.getSessionVars(tgUserID)
		_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, msg.MessageID))

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
				tgUserID,
				10*time.Second,
			)
		}

		if nameTaken {
			t := fmt.Sprintf("Portfolio with name '%s' already exists, try another name.", pName)
			return s.sendTemporaryMessage(
				tgbotapi.NewMessage(msg.Chat.ID,
					t),
				tgUserID,
				10*time.Second,
			)
		}

		s.sessions.setTempField(tgUserID, "TempPortfolioName", pName)
		s.sessions.setState(tgUserID, "waiting_portfolio_description")

		t := fmt.Sprintf("Please enter description for portfolio: %s", pName)
		return s.editMessageText(msg.Chat.ID, p.BotMessageID, t)

	case "waiting_portfolio_description":
		p, _ := s.sessions.getSessionVars(tgUserID)
		_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, p.BotMessageID))
		_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, msg.MessageID))

		portfolioDesc := msg.Text

		err = s.store.CreatePortfolio(ctx, dbUserID, p.TempPortfolioName, portfolioDesc)
		if err != nil {
			return fmt.Errorf("failed to create portfolio: %w", err)
		}

		s.sessions.clearSession(tgUserID)

		return s.sendTemporaryMessage(tgbotapi.NewMessage(msg.Chat.ID,
			"Portfolio created successfully!"),
			tgUserID,
			10*time.Second)

	// TODO: investigate it (delete)
	case "who_am_i":
		return nil
	}

	return nil
}
