package telegram_bot

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.com/avolkov/wood_post/pkg/log"
	t "gitlab.com/avolkov/wood_post/pkg/types"
)

func (s *Service) checkBeforeCreatePortfolio(
	ctx context.Context,
	chatID, tgUserID, dbUserID int64,
) error {
	r, ok := s.sessions.getSessionVars(tgUserID)
	if !ok {
		return nil
	}

	_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(chatID, r.BotMessageID))

	limitReached, err := s.store.ReachedPortfolioLimit(ctx, dbUserID)
	if err != nil {
		log.Errorf("could not check portfolios amount: %s", err)
		return s.sendTemporaryMessage(
			tgbotapi.NewMessage(
				chatID,
				"Oh, we could not create portfolio for you, please try again."),
			tgUserID,
			20*time.Second,
		)
	}
	log.Infof("user_id: %d, portfolios limit reached: %t", dbUserID, limitReached)

	if limitReached {
		err := s.sendTemporaryMessage(
			tgbotapi.NewMessage(chatID, "Sorry, you can create up to 2 portfolios. Gimmi ur munney to create more portfolios oi."),
			tgUserID,
			20*time.Second,
		)
		// err := s.editMessageText(
		// 	chatID,
		// 	r.BotMessageID,
		// 	"Sorry, you can create up to 2 portfolios. Gimmi ur munney to create more portfolios oi.")
		if err != nil {
			return err
		}
		return s.showMainMenu(chatID, tgUserID)
	}

	s.sessions.setState(tgUserID, "waiting_portfolio_name")

	// err := s.editMessageText(asdasdasd
	// 	chatID,
	// 	r.BotMessageID,
	// 	"Please enter a name for your portfolio without special characters:")

	msg := tgbotapi.NewMessage(chatID, "Please enter a name for your portfolio without special characters:")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Back", "cancel_action"),
		),
	)

	return s.sendTemporaryMessage(msg, tgUserID, 20*time.Second)
}

func (s *Service) ShowPortfolios(
	ctx context.Context,
	chatID, tgUserID, dbUserID int64,
	BotMsgID int,
	action string,
) error {
	_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(chatID, BotMsgID))

	onlyNonDefault := (action == "change_default")

	ps, err := s.store.GetPortfoliosFiltered(ctx, dbUserID, onlyNonDefault)
	if err != nil {
		log.Errorf("could not show portfolios: %s", err)
		return s.sendTemporaryMessage(
			tgbotapi.NewMessage(chatID,
				"Sorry, we cannot get your portfolios, please try again."),
			tgUserID,
			20*time.Second,
		)
	}

	if len(ps) == 0 {
		msg := tgbotapi.NewMessage(chatID, "You have no another portfolio, let's create a new one!")
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("New portfolio", "create_portfolio"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Back", "cancel_action"),
			),
		)
		return s.sendTemporaryMessage(msg, tgUserID, 20*time.Second)
	}

	log.Infof("user_id: %d, portfolios list: %s", dbUserID, ps)

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, p := range ps {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(p, fmt.Sprintf("%s::%s", action, p)),
		))
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Back", "cancel_action"),
	))

	msg := tgbotapi.NewMessage(chatID, "Select a portfolio to perform an action:")
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)

	// TODO: add Back button

	return s.sendTemporaryMessage(msg, tgUserID, 20*time.Second)
}

func (s *Service) askPortfolioConfirmation(
	chatID, tgUserID int64,
	BotMsgID int,
	nextAction string,
	args ...interface{},
) error {
	deleteMsg := tgbotapi.NewDeleteMessage(chatID, BotMsgID)
	_, _ = s.bot.Request(deleteMsg)

	template, ok := t.ConfirmationTemplates[nextAction]
	if !ok {
		return fmt.Errorf("unknown confirmation template for action: %s", nextAction)
	}

	text := fmt.Sprintf(template.MessageText, args...)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(template.ConfirmText, template.ConfirmCallback),
			tgbotapi.NewInlineKeyboardButtonData(template.CancelText, template.CancelCallback),
		),
	)

	s.sessions.setState(tgUserID, template.NextState)

	return s.sendTemporaryMessage(msg, tgUserID, 20*time.Second)
}

func (s *Service) portfolioRenameConfirmed(
	ctx context.Context,
	chatID, tgUserID, dbUserID int64,
	BotMsgID int,
	oldName, newName string,
) error {
	err := s.store.RenamePortfolio(ctx, dbUserID, oldName, newName)
	if err != nil {
		err := s.editMessageText(
			chatID,
			BotMsgID,
			"Could not rename portfolio, please try again.")
		if err != nil {
			return nil // ignore error cause we need to return db error
		}
		return err
	}

	log.Infof("portfolio renamed: user_id=%d, old_portfolio_name=%s, new_portfolio_name=%s", dbUserID, oldName, newName)

	err = s.editMessageText(
		chatID,
		BotMsgID,
		"Portfolio renamed successfully.")
	if err != nil {
		return err
	}
	return s.showMainMenu(chatID, tgUserID)
}

func (s *Service) portfolioDeletinonConfirmed(
	ctx context.Context,
	chatID, tgUserID, dbUserID int64,
	BotMsgID int,
	pName string,
) error {
	err := s.editMessageText(
		chatID,
		BotMsgID,
		"Deleting portfolio...")
	if err != nil {
		return err
	}

	time.Sleep(1 * time.Second)

	err = s.store.DeletePortfolio(ctx, dbUserID, pName)
	if err != nil {
		err := s.editMessageText(
			chatID,
			BotMsgID,
			"Could not delete portfolio, please try again.")
		if err != nil {
			return nil // ignore error cause we need to return db error
		}
		return err
	}

	log.Infof("portfolio deleted: user_id=%d, portfolio_name=%s", dbUserID, pName)

	err = s.editMessageText(chatID, BotMsgID, "Portfolio deleted successfully.")
	if err != nil {
		return err
	}
	return s.showMainMenu(chatID, tgUserID)
}

func (s *Service) portfolioChangeDefaultConfirmed(
	ctx context.Context,
	chatID, tgUserID, dbUserID int64,
	BotMsgID int,
	pName string,
) error {
	err := s.store.ChangeDefaultPortfolio(ctx, dbUserID, pName)
	if err != nil {
		err := s.editMessageText(
			chatID,
			BotMsgID,
			"Could not change default portfolio, please try again.")
		if err != nil {
			return nil // ignore error cause we need to return db error
		}
		return err
	}

	log.Infof("default portfolio changed: user_id=%d, portfolio_name=%s", dbUserID, pName)

	err = s.editMessageText(
		chatID,
		BotMsgID,
		"Default portfolio changed successfully.")
	if err != nil {
		return err
	}
	return s.showMainMenu(chatID, tgUserID)
}

func (s *Service) gfPortfoliosMain(chatID, tgUserID int64, BotMsgID int) error {
	_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(chatID, BotMsgID))

	actions := []t.Actiontype{
		{TgText: "New portfolio", CallBackName: "create_portfolio"}, // already exists
		{TgText: "General Report", CallBackName: "gf_portfolio_general_report"},
		{TgText: "Delete portfolio", CallBackName: "gf_portfolios_delete"},
		{TgText: "Get default", CallBackName: "gf_portfolio_get_default"},
		{TgText: "Change default", CallBackName: "gf_portfolio_change_default"},
		{TgText: "Rename", CallBackName: "gf_portfolio_rename"},
		{TgText: "Back to main menu", CallBackName: "cancel_action"},
	}

	var rows [][]tgbotapi.InlineKeyboardButton

	for i := 0; i < len(actions); i += 2 {
		row := []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(actions[i].TgText, actions[i].CallBackName),
		}
		if i+1 < len(actions) {
			row = append(row, tgbotapi.NewInlineKeyboardButtonData(actions[i+1].TgText, actions[i+1].CallBackName))
		}
		rows = append(rows, row)
	}

	msg := tgbotapi.NewMessage(chatID, "Choose an action:")
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)

	return s.sendTemporaryMessage(msg, tgUserID, 20*time.Second)
}

func (s *Service) performActionForPortfolio(
	ctx context.Context,
	chatID, tgUserID, dbUserID int64,
	BotMsgID int,
	cb string,
) error {
	name, err := s.store.GetDefaultPortfolio(ctx, dbUserID)
	if err != nil {
		return err
	}

	parts := strings.Split(cb, "::")
	action := parts[0]
	portfolio := parts[1]

	s.sessions.setTempField(tgUserID, "SelectedPortfolioName", portfolio)

	switch action {
	case "delete":
		if portfolio == name {
			_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(chatID, BotMsgID))

			msg := tgbotapi.NewMessage(chatID,
				fmt.Sprintf("You cannot delete *default* portfolio '*%s*'. Change default one first.", portfolio))
			msg.ParseMode = "Markdown"
			msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("Change default", "gf_portfolio_change_default"),
					tgbotapi.NewInlineKeyboardButtonData("Cancel", "cancel_action"),
				),
			)

			return s.sendTemporaryMessage(msg, tgUserID, 20*time.Second)
		}

		// return s.askDeletePortfolioConfirmation(chatID, tgUserID, BotMsgID, portfolio)
		return s.askPortfolioConfirmation(chatID, tgUserID, BotMsgID, "delete_portfolio", portfolio)

	case "rename":
		_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(chatID, BotMsgID))
		s.sessions.setState(tgUserID, "waiting_for_new_portfolio_name")
		msg := tgbotapi.NewMessage(
			chatID,
			fmt.Sprintf("Please enter a new name for portfolio *'%s'* without special characters.", portfolio))
		msg.ParseMode = "Markdown"
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Back", "cancel_action"),
			),
		)

		return s.sendTemporaryMessage(msg, tgUserID, 20*time.Second)

	case "change_default":
		_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(chatID, BotMsgID))
		return s.askPortfolioConfirmation(chatID, tgUserID, BotMsgID, "change_default_portfolio", portfolio)

	default:
		log.Errorf("invalid action in performActionForPortfolio: %s", err)

		return s.sendTemporaryMessage(
			tgbotapi.NewMessage(chatID, "Ops, something went wrong. please try again."),
			tgUserID, 20*time.Second)
	}
}

func (s *Service) prettyPortfolioName(portfolioName string) string {
	r := regexp.MustCompile(`[^a-zA-Z0-9_]+`)
	pName := r.ReplaceAllString(strings.ReplaceAll(portfolioName, " ", "_"), "")

	ru := regexp.MustCompile(`_+`)
	pName = ru.ReplaceAllString(strings.ToLower(pName), "_")

	if len(pName) > 40 {
		pName = pName[:40]
	}

	return pName
}

func (s *Service) showDefaultPortfolio(
	ctx context.Context,
	chatID, tgUserID, dbUserID int64,
	BotMsgID int,
) error {
	pName, err := s.store.GetDefaultPortfolio(ctx, dbUserID)
	if err != nil {
		// return err
		_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(chatID, BotMsgID))
		msg := tgbotapi.NewMessage(chatID, "You have no default portfolio yet. Let's add your first one!")
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("New portfolio", "create_portfolio"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Back to main menu", "cancel_action"),
			),
		)
		return s.sendTemporaryMessage(msg, tgUserID, 30*time.Second)
	}

	_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(chatID, BotMsgID))

	msg := tgbotapi.NewMessage(chatID,
		fmt.Sprintf("Your default portfolio name is *%s*.", pName))
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Change default", "gf_portfolio_change_default"),
			tgbotapi.NewInlineKeyboardButtonData("Rename", fmt.Sprintf("rename::%s", pName)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Back", "cancel_action"),
		),
	)

	return s.sendTemporaryMessage(msg, tgUserID, 20*time.Second)
}

func (s *Service) waitPortfolionName(
	ctx context.Context,
	chatID, tgUserID, dbUserID int64,
	BotMsgID int,
	msgText string,
) error {
	_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(chatID, BotMsgID))

	pName := s.prettyPortfolioName(msgText)

	nameTaken, err := s.store.PortfolioNameExists(ctx, dbUserID, pName)
	if err != nil {
		log.Errorf("could not check PortfolioNameExists: %s", err)
		return s.sendTemporaryMessage(
			tgbotapi.NewMessage(chatID,
				"Oh, we could not create portfolio for you, please try again."),
			tgUserID, 20*time.Second)
	}

	if nameTaken {
		t := fmt.Sprintf("Portfolio with name '%s' already exists, try another name.", pName)
		return s.sendTemporaryMessage(tgbotapi.NewMessage(chatID, t),
			tgUserID, 20*time.Second)
	}

	s.sessions.setTempField(tgUserID, "TempPortfolioName", pName)
	s.sessions.setState(tgUserID, "waiting_portfolio_description")

	// t := fmt.Sprintf("Please enter description for portfolio: %s", pName)asdasdasd
	// return s.editMessageText(chatID, BotMsgID, t)
	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Please enter description for portfolio: %s", pName))
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Back", "cancel_action"),
		),
	)
	return s.sendTemporaryMessage(msg, tgUserID, 20*time.Second)
}

func (s *Service) waitPortfolionDescription(
	ctx context.Context,
	chatID, tgUserID, dbUserID int64,
	BotMsgID int,
	portfolioName, msgText string,
) error {
	_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(chatID, BotMsgID))
	portfolioDesc := msgText

	err := s.store.CreatePortfolio(ctx, dbUserID, portfolioName, portfolioDesc)
	if err != nil {
		return fmt.Errorf("failed to create portfolio: %w", err)
	}

	s.sessions.clearSession(tgUserID)

	err = s.sendTemporaryMessage(
		tgbotapi.NewMessage(
			chatID,
			fmt.Sprintf("Portfolio '%s' created successfully!", portfolioName)),
		tgUserID,
		20*time.Second)
	if err != nil {
		return err
	}
	return s.showMainMenu(chatID, tgUserID)
}

func (s *Service) waitNewPortfolionName(
	ctx context.Context,
	chatID, tgUserID, dbUserID int64,
	BotMsgID int,
	SelectedPortfolioName, msgText string,
) error {
	_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(chatID, BotMsgID))
	pName := s.prettyPortfolioName(msgText)

	nameTaken, err := s.store.PortfolioNameExists(ctx, dbUserID, pName)
	if err != nil {
		return fmt.Errorf("failed to check portfolio existence: %w", err)
	}

	if nameTaken {
		msg := fmt.Sprintf("Portfolio with name '%s' already exists, try another name.", pName)
		return s.sendTemporaryMessage(
			tgbotapi.NewMessage(chatID, msg),
			tgUserID, 20*time.Second)
	}

	s.sessions.setTempField(tgUserID, "TempPortfolioName", pName)

	return s.askPortfolioConfirmation(
		chatID,
		tgUserID,
		BotMsgID,
		"rename_portfolio",
		SelectedPortfolioName,
		pName)
}

// showPortfolioGeneralReport displays a summary of all portfolios with their assets
func (s *Service) showPortfolioGeneralReport(
	ctx context.Context,
	chatID, tgUserID, dbUserID int64,
	BotMsgID int,
) error {
	_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(chatID, BotMsgID))

	summaries, err := s.store.GetPortfolioSummariesForUser(ctx, dbUserID)
	if err != nil {
		log.Error("Failed to get portfolio summaries", "error", err, "user_id", dbUserID)
		return s.sendTemporaryMessage(
			tgbotapi.NewMessage(chatID, "Sorry, couldn't retrieve your portfolio data. Please try again."),
			tgUserID, 20*time.Second)
	}

	if len(summaries) == 0 {
		msg := tgbotapi.NewMessage(chatID, "You don't have any portfolios with assets yet. Start by creating a portfolio and adding some transactions!")
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("New portfolio", "create_portfolio"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Back", "cancel_action"),
			),
		)
		return s.sendTemporaryMessage(msg, tgUserID, 30*time.Second)
	}

	// Build the report message
	var reportText strings.Builder
	reportText.WriteString("*📊 PORTFOLIO GENERAL REPORT*\n\n")

	var grandTotalUSD float64
	for i, summary := range summaries {
		if i > 0 {
			reportText.WriteString("\n")
		}

		reportText.WriteString(fmt.Sprintf("*Portfolio: %s*\n", summary.Name))

		var portfolioTotalUSD float64
		for _, asset := range summary.Assets {
			// Extract base asset from pair (e.g., BTC from BTCUSDT)
			baseCurrency := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(asset.Pair, "USDT"), "USDC"), "USD"), "EUR")

			reportText.WriteString(fmt.Sprintf("%s: %.6g %s, %.2f USD\n",
				asset.Pair,
				asset.TotalAmount,
				baseCurrency,
				asset.TotalUSD))

			portfolioTotalUSD += asset.TotalUSD
		}

		reportText.WriteString(fmt.Sprintf("*Portfolio Total: %.2f USD*\n", portfolioTotalUSD))
		grandTotalUSD += portfolioTotalUSD

		if i < len(summaries)-1 {
			reportText.WriteString("─────────────────────\n")
		}
	}

	reportText.WriteString(fmt.Sprintf("\n*🎯 GRAND TOTAL: %.2f USD*", grandTotalUSD))

	msg := tgbotapi.NewMessage(chatID, reportText.String())
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Refresh", "gf_portfolio_general_report"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Back to Portfolios", "portfolios"),
			tgbotapi.NewInlineKeyboardButtonData("Main Menu", "cancel_action"),
		),
	)

	return s.sendTemporaryMessage(msg, tgUserID, 60*time.Second)
}
