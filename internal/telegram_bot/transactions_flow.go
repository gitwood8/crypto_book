package telegram_bot

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.com/avolkov/wood_post/pkg/log"
	t "gitlab.com/avolkov/wood_post/pkg/types"
)

func (s *Service) gfTransactionsMain(chatID, tgUserID int64, BotMsgID int) error {
	_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(chatID, BotMsgID))

	actions := []t.Actiontype{
		{TgText: "Add transaction", CallBackName: "gf_add_transaction"},
		{TgText: "Show last 5 added transactions", CallBackName: "gf_show_last_5_transactions"},
		{TgText: "Delete transaction", CallBackName: "gf_delete_transaction"},
		// {TgText: "Change default", CallBackName: "gf_portfolio_change_default"},
		// {TgText: "Rename", CallBackName: "gf_portfolio_rename"},
		{TgText: "Back to main menu", CallBackName: "cancel_action"},
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, a := range actions {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(a.TgText, a.CallBackName),
		))
	}

	msg := tgbotapi.NewMessage(chatID, "Choose an action:")
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)

	return s.sendTemporaryMessage(msg, tgUserID, 20*time.Second)
}

func (s *Service) askTransactionType(
	ctx context.Context,
	chatID, tgUserID, dbUserID int64,
	BotMsgID int,
) error {
	_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(chatID, BotMsgID))

	exists, err := s.store.PortfolioExists(ctx, dbUserID)
	if err != nil {
		return err
	}

	if !exists {
		msg := tgbotapi.NewMessage(chatID, "You have no portfolios yet. Let's create a new one!")
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

	msg := tgbotapi.NewMessage(chatID,
		"Choose what type of transaction do you want to add:")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Buy", "tx_type_buy"),
			tgbotapi.NewInlineKeyboardButtonData("Sell", "tx_type_sell"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Main menu", "cancel_action"),
		),
	)

	s.sessions.setState(tgUserID, "waiting_transaction_type")
	return s.sendTemporaryMessage(msg, tgUserID, 20*time.Second)
}

func (s *Service) askTransactionAsset(
	ctx context.Context,
	chatID, tgUserID, dbUserID int64,
	BotMsgID int,
	txData *t.TempTransactionData,
	txType string,
) error {
	_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(chatID, BotMsgID))

	log.Info("raw tx type: ", txType)

	txTypeClean := strings.TrimPrefix(txType, "tx_type_")
	txData.Type = txTypeClean

	log.Info("chosen tx type: ", txTypeClean)

	msg := tgbotapi.NewMessage(chatID,
		"Please choose an asset ticker or enter a new one (e.g. BTC, eth, DoGe).")
	msg.ParseMode = "Markdown"

	var topAssets []string

	topAssets, err := s.store.GetTopAssetsForUser(ctx, dbUserID)
	if err != nil {
		return err
	}

	defaultAssets := t.DefaultCryptoPairs

	allAssets := s.mergeUniqueAssets(defaultAssets, topAssets)

	var rows [][]tgbotapi.InlineKeyboardButton
	for i := 0; i < len(allAssets); i += 2 {
		row := []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(allAssets[i], "tx_asset_chosen_"+allAssets[i]),
		}
		if i+1 < len(allAssets) {
			row = append(row, tgbotapi.NewInlineKeyboardButtonData(allAssets[i+1], "tx_asset_chosen_"+allAssets[i+1]))
		}
		rows = append(rows, row)
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Back", "gf_add_transaction"),
	))

	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)

	s.sessions.setState(tgUserID, "waiting_transaction_asset")
	return s.sendTemporaryMessage(msg, tgUserID, 20*time.Second)
}

func (s *Service) askTransactionAssetAmount(
	chatID, tgUserID int64,
	BotMsgID int,
	msgText string,
	txData *t.TempTransactionData,
) error {
	selectedAsset := strings.TrimPrefix(msgText, "tx_asset_chosen_")

	_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(chatID, BotMsgID))

	result, err := s.handleTransactionValidationError(selectedAsset, "asset", chatID, tgUserID)
	if err != nil {
		return err
	}

	txData.Asset = result.(string)

	msg := tgbotapi.NewMessage(chatID,
		"Enter the asset amount (e.g. 1234, 12.34).")
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Back", "gf_add_transaction"),
		),
	)

	s.sessions.setState(tgUserID, "waiting_transaction_asset_amount")
	return s.sendTemporaryMessage(msg, tgUserID, 20*time.Second)
}

func (s *Service) askTransactionAssetPrice(
	chatID, tgUserID int64,
	BotMsgID int,
	msgText string,
	txData *t.TempTransactionData,
) error {
	fmt.Println("amount", msgText)
	_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(chatID, BotMsgID))

	result, err := s.handleTransactionValidationError(msgText, "amount", chatID, tgUserID)
	if err != nil {
		return err
	}

	txData.AssetAmount = result.(float64)

	msg := tgbotapi.NewMessage(chatID,
		"Enter the asset price (e.g. bought BTC for a 15500 usdt).")
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Back", "gf_add_transaction"),
		),
	)

	s.sessions.setState(tgUserID, "waiting_transaction_asset_price")
	return s.sendTemporaryMessage(msg, tgUserID, 20*time.Second)
}

func (s *Service) askTransactionDate(
	chatID, tgUserID int64,
	BotMsgID int,
	msgText string,
	txData *t.TempTransactionData,
) error {
	fmt.Println("price", msgText)

	_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(chatID, BotMsgID))

	result, err := s.handleTransactionValidationError(msgText, "price", chatID, tgUserID)
	if err != nil {
		return err
	}

	fmt.Printf("priceFloat: %s\n", reflect.TypeOf(result))

	txData.AssetPrice = result.(float64)

	msg := tgbotapi.NewMessage(chatID,
		"Select transaction date or enter manually in format *YYYY-MM-DD* (e.g. 2025-06-15):")
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Today", "tx_date_today"),
			tgbotapi.NewInlineKeyboardButtonData("Yesterday", "tx_date_yesterday"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("2 days ago", "tx_date_2days"),
			tgbotapi.NewInlineKeyboardButtonData("1 week ago", "tx_date_1week"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("1 month ago", "tx_date_1month"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Back", "gf_add_transaction"),
		),
	)

	s.sessions.setState(tgUserID, "waiting_transaction_date")
	return s.sendTemporaryMessage(msg, tgUserID, 20*time.Second)
}

func (s *Service) asktransactionConfirmation(
	chatID, tgUserID int64,
	BotMsgID int,
	dateString string,
	txData *t.TempTransactionData,
) error {

	_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(chatID, BotMsgID))

	log.Info("date raw", dateString)
	dateValue := strings.TrimPrefix(dateString, "tx_date_")
	log.Info("date", dateValue)

	result, err := s.handleTransactionValidationError(dateValue, "date", chatID, tgUserID)
	if err != nil {
		return err
	}

	fmt.Printf("dateTime: %s\n", reflect.TypeOf(result))

	txData.TransactionDate = result.(time.Time)

	txData.USDAmount =
		txData.AssetAmount * txData.AssetPrice

	// Old table format
	// tableText := fmt.Sprintf(
	// 	"*You are about to add a new transaction. Please confirm:*\n\n"+
	// 		"```\n"+
	// 		"| %-12s | %-8s | %-13s | %-11s | %-10s | %-10s |\n"+
	// 		"|--------------+----------+---------------+-------------+------------+------------|\n"+
	// 		"| %-12s | %-8s | %-13.4f | %-11.4f | %-10.2f | %-10s |\n"+
	// 		"```",
	// 	"Pair", "Type", "Asset Amount", "Asset Price", "USD Amount", "Date",
	// 	txData.Pair,
	// 	strings.ToUpper(txData.Type),
	// 	txData.AssetAmount,
	// 	txData.AssetPrice,
	// 	txData.USDAmount,
	// 	txData.TransactionDate.Format("2006-01-02"),
	// )

	var typeEmoji string
	switch strings.ToLower(txData.Type) {
	case "buy":
		typeEmoji = "🟢"
	case "sell":
		typeEmoji = "🔴"
	default:
		typeEmoji = "🔵"
	}

	// portfolioID, err := s.store.GetDefaultPortfolioID(ctx, dbUserID)
	// if err != nil {
	// 	return err
	// }

	// New simplified format
	tableText := fmt.Sprintf(
		"*You are about to add a new transaction. Please confirm:*\n\n"+
			"%s *%s %s*\n"+
			"Type: `%s`\n"+
			"Amount: `%.8g %s`\n"+
			"Price: `$%.2f`\n"+
			"Total: `$%.2f`\n"+
			"Date: `%s`\n",
		typeEmoji,
		strings.ToUpper(txData.Type),
		txData.Asset, // FIXME check if its correct
		strings.ToUpper(txData.Type),
		txData.AssetAmount,
		txData.Asset,
		txData.AssetPrice,
		txData.USDAmount,
		txData.TransactionDate.Format("2006-01-02"),
	)

	msg := tgbotapi.NewMessage(chatID, tableText)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Confirm", "tx_confirm_transaction"),
			tgbotapi.NewInlineKeyboardButtonData("Back", "gf_add_transaction"),
		),
	)
	msg.ParseMode = "Markdown"

	s.sessions.setState(tgUserID, "waiting_transaction_confirmation")
	return s.sendTemporaryMessage(msg, tgUserID, 20*time.Second)
}

func (s *Service) transactionConfirmed(
	ctx context.Context,
	chatID, tgUserID, dbUserID int64,
	BotMsgID int,
	txData *t.TempTransactionData,
) error {
	_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(chatID, BotMsgID))

	portfolioID, err := s.store.GetDefaultPortfolioID(ctx, dbUserID)
	if err != nil {
		return err
	}

	err = s.store.AddNewTransaction(ctx, dbUserID, portfolioID, txData)
	if err != nil {
		return err
	}
	err = s.sendTemporaryMessage(
		tgbotapi.NewMessage(
			chatID,
			fmt.Sprintf("Transaction added successfully: %s, %.2f USD!", txData.Asset, txData.USDAmount)),
		tgUserID,
		10*time.Second)
	if err != nil {
		return err
	}

	log.Info("transaction added successfully", "user_id", dbUserID)

	return s.showMainMenu(chatID, tgUserID)
}

func (s *Service) transactionValidateInput(rawText string, inputType string) (any, error) {
	text := strings.TrimSpace(rawText)

	switch inputType {
	case "asset":
		cleaned := strings.ToUpper(strings.TrimSpace(text)) // FIXME test this

		// Asset ticker validation: 3-8 characters, only letters
		validRe := regexp.MustCompile(`^[A-Z]{3,8}$`)
		if !validRe.MatchString(cleaned) {
			log.Warnf("invalid asset format: %s", cleaned)
			return "Wrong asset format. Use asset ticker like 'BTC', 'ETH', 'DOGE'. Only letters allowed, 3-8 characters total.",
				fmt.Errorf("invalid asset format")
		}

		return cleaned, nil

	case "amount":
		if !regexp.MustCompile(`^\d+(\.\d{1,8})?$`).MatchString(text) {
			return "Wrong amount format. Use numbers with up to 8 decimal places (e.g. 1234, 12.345, 0.00000001).",
				fmt.Errorf("invalid amount format")
		}

		val, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return "Could not parse amount. Please try again.", err
		}

		if val <= 0 {
			return "Amount must be greater than 0.", fmt.Errorf("amount must be positive")
		}
		if val > 1000000000 {
			return "Amount too large. Maximum allowed: 1,000,000,000.", fmt.Errorf("amount too large")
		}
		if val < 0.00000001 {
			return "Amount too small. Minimum allowed: 0.00000001.", fmt.Errorf("amount too small")
		}

		return val, nil

	case "price":
		if !regexp.MustCompile(`^\d+(\.\d{1,8})?$`).MatchString(text) {
			return "Wrong price format. Use numbers with up to 8 decimal places (e.g. 1234, 12.345, 0.00000001).",
				fmt.Errorf("invalid price format")
		}

		//FIXME add sending message to tg
		val, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return "Could not parse price. Please try again.", err
		}

		if val <= 0 {
			return "Price must be greater than 0.", fmt.Errorf("price must be positive")
		}
		if val > 10000000 {
			return "Price too high. Maximum allowed: 10,000,000.", fmt.Errorf("price too high")
		}
		if val < 0.00000001 {
			return "Price too small. Minimum allowed: 0.00000001.", fmt.Errorf("price too small")
		}

		return val, nil

		// FIXME week and month looks unnecessary

	case "date":
		now := time.Now()
		switch strings.ToLower(text) {
		case "today":
			return now, nil
		case "yesterday":
			return now.AddDate(0, 0, -1), nil
		case "2days", "2 days ago":
			return now.AddDate(0, 0, -2), nil
		case "1week", "1 week ago":
			return now.AddDate(0, 0, -7), nil
		case "1month", "1 month ago":
			return now.AddDate(0, -1, 0), nil
		}

		if !regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`).MatchString(text) {
			return "Wrong date format. Use YYYY-MM-DD (e.g. 2024-06-14) or select a date button.",
				fmt.Errorf("invalid date format")
		}

		parsedTime, err := time.Parse("2006-01-02", text)
		if err != nil {
			return "Could not parse the date. Please use YYYY-MM-DD format.", err
		}

		if parsedTime.After(now.AddDate(0, 0, 1)) {
			return "Transaction date cannot be in the future.", fmt.Errorf("future date not allowed")
		}
		if parsedTime.Before(now.AddDate(-10, 0, 0)) { // 10 years ago
			return "Transaction date is too old (maximum 10 years ago).", fmt.Errorf("date too old")
		}

		return parsedTime, nil

	default:
		return nil, fmt.Errorf("unknown input type: %s", inputType)
	}
}

// helper method to handle validation errors consistently
// It validates input and sends error message if validation fails
func (s *Service) handleTransactionValidationError(
	msgText, inputType string,
	chatID, tgUserID int64,
) (any, error) {
	result, err := s.transactionValidateInput(msgText, inputType)
	if err != nil {
		errorText := result.(string)
		msg := tgbotapi.NewMessage(chatID, errorText)
		msg.ParseMode = "Markdown"
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Try again", "gf_add_transaction"),
			),
		)
		sendErr := s.sendTemporaryMessage(msg, tgUserID, 20*time.Second)
		if sendErr != nil {
			return nil, sendErr
		}
		// Return the original validation error, not the send error
		return nil, err
	}
	return result, nil
}

func (s *Service) mergeUniqueAssets(defaultAssets, topAssets []string) []string {
	unique := make(map[string]struct{})
	var result []string

	for _, asset := range append(defaultAssets, topAssets...) {
		if _, exists := unique[asset]; !exists {
			unique[asset] = struct{}{}
			result = append(result, asset)
		}
	}
	return result
}

func (s *Service) showLast5Transactions(
	ctx context.Context,
	chatID, tgUserID, dbUserID int64,
	BotMsgID int,
) error {
	_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(chatID, BotMsgID))

	transactions, err := s.store.GetLast5TransactionsForUser(ctx, dbUserID)
	if err != nil {
		log.Errorf("could not get last 5 transactions: %s", err)
		return s.sendTemporaryMessage(
			tgbotapi.NewMessage(chatID,
				"Sorry, we cannot get your transactions, please try again."),
			tgUserID,
			20*time.Second,
		)
	}

	if len(transactions) == 0 {
		msg := tgbotapi.NewMessage(chatID, "You have no transactions yet. Let's add your first one!")
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Add transaction", "gf_add_transaction"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Back to main menu", "cancel_action"),
			),
		)
		return s.sendTemporaryMessage(msg, tgUserID, 30*time.Second)
	}

	// format transactions in a user-friendly way
	var messageText strings.Builder
	messageText.WriteString("*Your Last 5 Transactions:*\n\n")

	for i, tx := range transactions {
		var typeEmoji string
		switch strings.ToLower(tx.Type) {
		case "buy":
			typeEmoji = "🟢"
		case "sell":
			typeEmoji = "🔴"
		default:
			typeEmoji = "🔵"
		}

		messageText.WriteString(fmt.Sprintf(
			"%s *%s %s*\n"+
				"Portfolio: `%s`\n"+
				"Amount: `%.8g %s`\n"+
				"Price: `$%.2f`\n"+
				"Total: `$%.2f`\n"+
				"Date: `%s`\n",
			typeEmoji,
			strings.ToUpper(tx.Type),
			tx.Asset, // FIXME check if its correct
			tx.PortfolioName,
			tx.AssetAmount,
			tx.Asset,
			tx.AssetPrice,
			tx.USDAmount,
			tx.TransactionDate.Format("2006-01-02"),
		))

		// add note if available
		if tx.Note != "" {
			messageText.WriteString(fmt.Sprintf("📝 Note: `%s`\n", tx.Note))
		}

		// add separator except for last transaction
		if i < len(transactions)-1 {
			messageText.WriteString("\n" + strings.Repeat("─", 17) + "\n\n")
		}
	}

	msg := tgbotapi.NewMessage(chatID, messageText.String())
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Add new transaction", "gf_add_transaction"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Back to transactions menu", "gf_transactions_main"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Back to main menu", "cancel_action"),
		),
	)

	return s.sendTemporaryMessage(msg, tgUserID, 40*time.Second)
}
