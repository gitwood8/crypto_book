package telegram_bot

import (
	"context"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
	"gitlab.com/avolkov/wood_post/pkg/log"
)

func (s *Service) handleCallback(ctx context.Context, cb *tgbotapi.CallbackQuery, sv *UserSession, tgUserID int64) error {
	// tgUserID := cb.From.ID
	// FIXME: check this log

	dbUserID, err := s.store.GetUserIDByTelegramID(ctx, tgUserID)
	if err != nil {
		return errors.Wrap(err, "failed to get user from DB")
	}

	log.Infof("user_id: %d, selected callback: %s", dbUserID, cb.Data)

	switch {
	case cb.Data == "create_portfolio":
		return s.checkBeforeCreatePortfolio(ctx, cb.Message.Chat.ID, tgUserID, dbUserID)

	case cb.Data == "who_am_i":
		return s.showServiceInfo(cb.Message.Chat.ID, tgUserID)

		// case cb.Data == "gf_portfolios":
		// 	return s.gfPortfoliosMain(cb.Message.Chat.ID, tgUserID, r.BotMessageID)

	case cb.Data == "gf_portfolios_main":
		return s.gfPortfoliosMain(cb.Message.Chat.ID, tgUserID, sv.BotMessageID)

	case cb.Data == "gf_portfolios_delete":
		s.sessions.setTempField(tgUserID, "NextAction", "delete")
		return s.ShowPortfolios(ctx, cb.Message.Chat.ID, tgUserID, dbUserID, sv.BotMessageID, "delete")

	case cb.Data == "gf_portfolio_rename":
		s.sessions.setTempField(tgUserID, "NextAction", "rename")
		return s.ShowPortfolios(ctx, cb.Message.Chat.ID, tgUserID, dbUserID, sv.BotMessageID, "rename")

	case cb.Data == "gf_portfolio_change_default":
		s.sessions.setTempField(tgUserID, "NextAction", "change_default")
		return s.ShowPortfolios(ctx, cb.Message.Chat.ID, tgUserID, dbUserID, sv.BotMessageID, "change_default")

	case cb.Data == "gf_portfolio_get_default":
		return s.showDefaultPortfolio(ctx, cb.Message.Chat.ID, tgUserID, dbUserID, sv.BotMessageID)

	// ----------- REPORTS -----------
	case cb.Data == "gf_reports_main":
		return s.gfReportsMain(cb.Message.Chat.ID, tgUserID, sv.BotMessageID)

	case cb.Data == "gf_reports_general":
		return s.showPortfolioGeneralReport(ctx, cb.Message.Chat.ID, tgUserID, dbUserID, sv.BotMessageID)

	// case cb.Data == "gf_reports_advanced":
	// 	return s.showPortfolioAdvancedReport(ctx, cb.Message.Chat.ID, tgUserID, dbUserID, sv.BotMessageID)

	// ----------- REPORTS -----------

	case strings.Contains(cb.Data, "::"):
		// log.Infof("callback data: %s", cb.Data)
		return s.performActionForPortfolio(ctx, cb.Message.Chat.ID, tgUserID, dbUserID, sv.BotMessageID, cb.Data)

	case cb.Data == "confirm_portfolio_deletion":
		return s.portfolioDeletinonConfirmed(ctx, cb.Message.Chat.ID, tgUserID, dbUserID, sv.BotMessageID, sv.SelectedPortfolioName)

	case cb.Data == "confirm_portfolio_rename":
		return s.portfolioRenameConfirmed(ctx, cb.Message.Chat.ID, tgUserID, dbUserID, sv.BotMessageID, sv.SelectedPortfolioName, sv.TempPortfolioName)

	case cb.Data == "confirm_portfolio_change_default":
		return s.portfolioChangeDefaultConfirmed(ctx, cb.Message.Chat.ID, tgUserID, dbUserID, sv.BotMessageID, sv.SelectedPortfolioName)

	// ------- TRANSACTIONS -------
	case cb.Data == "gf_transactions_main":
		return s.gfTransactionsMain(cb.Message.Chat.ID, tgUserID, sv.BotMessageID)

	case cb.Data == "gf_add_transaction":
		return s.askTransactionType(ctx, cb.Message.Chat.ID, tgUserID, dbUserID, sv.BotMessageID)

	case cb.Data == "gf_show_last_5_transactions":
		return s.showLast5Transactions(ctx, cb.Message.Chat.ID, tgUserID, dbUserID, sv.BotMessageID)

	case strings.Contains(cb.Data, "tx_type_"):
		return s.askTransactionPair(ctx, cb.Message.Chat.ID, tgUserID, dbUserID, sv.BotMessageID, &sv.TempTransaction, cb.Data)

	case cb.Data == "tx_confirm_transaction":
		return s.transactionConfirmed(ctx, cb.Message.Chat.ID, tgUserID, dbUserID, sv.BotMessageID, &sv.TempTransaction)

	case strings.Contains(cb.Data, "tx_pair_chosen_"):
		return s.askTransactionAssetAmount(cb.Message.Chat.ID, tgUserID, sv.BotMessageID, cb.Data, &sv.TempTransaction)

	case strings.Contains(cb.Data, "tx_date_"):
		return s.asktransactionConfirmation(cb.Message.Chat.ID, tgUserID, sv.BotMessageID, cb.Data, &sv.TempTransaction)

	case cb.Data == "cancel_action":
		s.sessions.clearSession(tgUserID)
		_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(cb.Message.Chat.ID, sv.BotMessageID))
		return s.showMainMenu(cb.Message.Chat.ID, tgUserID)
	}
	// fmt.Println("portfolio, callback: ", action, p)

	return nil
}

func (s *Service) handleMessage(ctx context.Context, msg *tgbotapi.Message, sv *UserSession, tgUserID int64) error {
	// tgUserID := msg.From.ID
	_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, msg.MessageID))
	state, ok := s.sessions.getState(tgUserID)
	if !ok {
		return nil
	}

	//TODO: i can add tgID to temp field to avoid db requests every time
	dbUserID, err := s.store.GetUserIDByTelegramID(ctx, msg.From.ID)
	if err != nil {
		return errors.Wrap(err, "failed to get user from DB")
	}

	switch state {
	case "waiting_portfolio_name":
		return s.waitPortfolionName(ctx, msg.Chat.ID, tgUserID, dbUserID, sv.BotMessageID, msg.Text)

	case "waiting_portfolio_description":
		return s.waitPortfolionDescription(ctx, msg.Chat.ID, tgUserID, dbUserID, sv.BotMessageID, sv.TempPortfolioName, msg.Text)

	case "waiting_for_new_portfolio_name":
		return s.waitNewPortfolionName(ctx, msg.Chat.ID, tgUserID, dbUserID, sv.BotMessageID, sv.SelectedPortfolioName, msg.Text)

	case "waiting_transaction_pair":
		return s.askTransactionAssetAmount(msg.Chat.ID, tgUserID, sv.BotMessageID, msg.Text, &sv.TempTransaction)

	case "waiting_transaction_asset_amount":
		return s.askTransactionAssetPrice(msg.Chat.ID, tgUserID, sv.BotMessageID, msg.Text, &sv.TempTransaction)

	case "waiting_transaction_asset_price":
		return s.askTransactionDate(msg.Chat.ID, tgUserID, sv.BotMessageID, msg.Text, &sv.TempTransaction)

		// TODO investigee do i need this if i have callback
	// case "waiting_transaction_date":
	// 	return s.asktransactionConfirmation(msg.Chat.ID, tgUserID, sv.BotMessageID, msg.Text, &sv.TempTransaction)

	case "main_menu":
		text := msg.Text

		switch text {
		case "My portfolios":
			log.Infof("main menu: %s", text)
			return s.gfPortfoliosMain(msg.Chat.ID, tgUserID, sv.BotMessageID)

		case "Transactions":
			log.Infof("main menu: %s", text)
			return s.gfTransactionsMain(msg.Chat.ID, tgUserID, sv.BotMessageID)

		case "Reports":
			log.Infof("main menu: %s", text)
			return s.gfReportsMain(msg.Chat.ID, tgUserID, sv.BotMessageID)

		case "Help":
			log.Infof("main menu: %s", text)
			return s.showServiceInfo(msg.Chat.ID, tgUserID)
		}
	}

	return nil
}
