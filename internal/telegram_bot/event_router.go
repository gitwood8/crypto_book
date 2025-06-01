package telegram_bot

import (
	"context"
	"fmt"
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
	// 	return s.sendTemporaryMessage(mainMenu, update.Message.From.ID, 15*time.Second)

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

	switch {
	case cb.Data == "create_portfolio":
		return s.checkBeforeCreatePortfolio(ctx, cb.Message.Chat.ID, tgUserID, dbUserID)
	// case "who_am_i":
	// TODO: will be added later

	// s.sessions.setState(tgUserID, "who_am_i") // why?
	// msg := tgbotapi.NewMessage(cb.Message.Chat.ID, "Im very cool bot")
	// return s.sendTemporaryMessage(msg, 15*time.Second)
	// ---------------------------------------------

	// case cb.Data == "gf_portfolios":
	// 	return s.gfPortfoliosMain(cb.Message.Chat.ID, tgUserID, r.BotMessageID)

	case cb.Data == "gf_portfolios_delete":
		s.sessions.setTempField(tgUserID, "NextAction", "delete")
		return s.ShowPortfolios(ctx, cb.Message.Chat.ID, tgUserID, dbUserID, r.BotMessageID, "delete")

	case cb.Data == "gf_portfolio_rename":
		s.sessions.setTempField(tgUserID, "NextAction", "rename")
		return s.ShowPortfolios(ctx, cb.Message.Chat.ID, tgUserID, dbUserID, r.BotMessageID, "rename")

	case cb.Data == "gf_portfolio_change_default":
		s.sessions.setTempField(tgUserID, "NextAction", "change_default")
		return s.ShowPortfolios(ctx, cb.Message.Chat.ID, tgUserID, dbUserID, r.BotMessageID, "change_default")

	case cb.Data == "gf_portfolio_get_default":
		return s.showDefaultPortfolio(ctx, cb.Message.Chat.ID, tgUserID, dbUserID, r.BotMessageID)

	case strings.Contains(cb.Data, "::"):
		log.Infof("callback data: %s", cb.Data)
		return s.performActionForPortfolio(ctx, cb.Message.Chat.ID, tgUserID, dbUserID, r.BotMessageID, cb.Data)

	case cb.Data == "confirm_portfolio_deletion":
		return s.portfolioDeletinonConfirmed(ctx, cb.Message.Chat.ID, tgUserID, dbUserID, r.BotMessageID, r.SelectedPortfolioName)

	case cb.Data == "confirm_portfolio_rename":
		return s.portfolioRenameConfirmed(ctx, cb.Message.Chat.ID, tgUserID, dbUserID, r.BotMessageID, r.SelectedPortfolioName, r.TempPortfolioName)

	case cb.Data == "confirm_portfolio_change_default":
		return s.portfolioChangeDefaultConfirmed(ctx, cb.Message.Chat.ID, tgUserID, dbUserID, r.BotMessageID, r.SelectedPortfolioName)

	case cb.Data == "cancel_action":
		s.sessions.clearSession(tgUserID)
		// return s.editMessageText(cb.Message.Chat.ID, r.BotMessageID, "")
		_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(cb.Message.Chat.ID, r.BotMessageID))
		return s.showMainMenu(cb.Message.Chat.ID, tgUserID)
	}
	// fmt.Println("portfolio, callback: ", action, p)

	return nil
}

func (s *Service) handleMessage(ctx context.Context, msg *tgbotapi.Message) error {
	tgUserID := msg.From.ID
	_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, msg.MessageID))
	state, ok := s.sessions.getState(tgUserID)
	if !ok {
		return nil
	}

	p, _ := s.sessions.getSessionVars(tgUserID)

	//TODO: i can add tgID to temp field to avoid db requests every time
	dbUserID, err := s.store.GetUserIDByTelegramID(ctx, msg.From.ID)
	if err != nil {
		return errors.Wrap(err, "failed to get user from DB")
	}

	switch state {
	case "waiting_portfolio_name":
		pName := s.prettyPortfolioName(msg.Text)

		nameTaken, err := s.store.PortfolioNameExists(ctx, dbUserID, pName)
		if err != nil {
			log.Errorf("could not check PortfolioNameExists: %s", err)
			return s.sendTemporaryMessage(
				tgbotapi.NewMessage(msg.Chat.ID,
					"Oh, we could not create portfolio for you, please try again."),
				tgUserID, 15*time.Second)
		}

		if nameTaken {
			t := fmt.Sprintf("Portfolio with name '%s' already exists, try another name.", pName)
			return s.sendTemporaryMessage(tgbotapi.NewMessage(msg.Chat.ID, t),
				tgUserID, 15*time.Second)
		}

		s.sessions.setTempField(tgUserID, "TempPortfolioName", pName)
		s.sessions.setState(tgUserID, "waiting_portfolio_description")

		t := fmt.Sprintf("Please enter description for portfolio: %s", pName)
		return s.editMessageText(msg.Chat.ID, p.BotMessageID, t)

	case "waiting_portfolio_description":
		_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, p.BotMessageID))
		portfolioDesc := msg.Text

		err = s.store.CreatePortfolio(ctx, dbUserID, p.TempPortfolioName, portfolioDesc)
		if err != nil {
			return fmt.Errorf("failed to create portfolio: %w", err)
		}

		s.sessions.clearSession(tgUserID)

		err := s.sendTemporaryMessage(tgbotapi.NewMessage(msg.Chat.ID,
			"Portfolio created successfully!"), tgUserID, 10*time.Second)
		if err != nil {
			return err
		}
		return s.showMainMenu(msg.Chat.ID, tgUserID)

	case "waiting_for_new_portfolio_name":
		_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, p.BotMessageID))
		pName := s.prettyPortfolioName(msg.Text)
		s.sessions.setTempField(tgUserID, "TempPortfolioName", pName)

		return s.askPortfolioConfirmation(msg.Chat.ID, tgUserID, p.BotMessageID, "rename_portfolio", p.SelectedPortfolioName, pName)

	case "main_menu":
		text := msg.Text

		switch text {
		case "My portfolios":
			log.Infof("main menu: %s", text)
			return s.gfPortfoliosMain(msg.Chat.ID, tgUserID, p.BotMessageID)

		case "Transactions":
			log.Infof("main menu: %s", text)
			return nil
		}
	}

	return nil
}
