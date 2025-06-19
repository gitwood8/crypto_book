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
	t "gitlab.com/avolkov/wood_post/pkg/types"
)

func (s *Service) gfTransactionsMain(chatID, tgUserID int64, BotMsgID int) error {
	_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(chatID, BotMsgID))

	actions := []t.Actiontype{
		{TgText: "Add transaction", CallBackName: "gf_add_transaction"},
		{TgText: "Show last 5 added transactions", CallBackName: "gf_portfolios_delete"},
		// {TgText: "Get default", CallBackName: "gf_portfolio_get_default"},
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

func (s *Service) askTransactionPair(
	ctx context.Context,
	chatID, tgUserID, dbUserID int64,
	BotMsgID int,
) error {
	_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(chatID, BotMsgID))

	msg := tgbotapi.NewMessage(chatID,
		"Please enter a currency pair (e.g. BTCUSDT, ETHUSDT etc. 'btc usdt' - also fine).")
	msg.ParseMode = "Markdown"

	var topPairs []string

	topPairs, err := s.store.GetTopPairsForUser(ctx, dbUserID)
	if err != nil {
		return err
	}

	defaultPairs := []string{"BTCUSDT", "ETHUSDT", "DOGEUSDT"}

	allPairs := s.mergeUniqueTxPairs(defaultPairs, topPairs)

	var rows [][]tgbotapi.InlineKeyboardButton
	for i := 0; i < len(allPairs); i += 2 {
		row := []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(allPairs[i], "tx_pair_chosen_"+allPairs[i]),
		}
		if i+1 < len(allPairs) {
			row = append(row, tgbotapi.NewInlineKeyboardButtonData(allPairs[i+1], "tx_pair_chosen_"+allPairs[i+1]))
		}
		rows = append(rows, row)
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Cancel", "cancel_action"),
	))

	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)

	s.sessions.setState(tgUserID, "waiting_transaction_pair")
	return s.sendTemporaryMessage(msg, tgUserID, 20*time.Second)
}

func (s *Service) askTransactionAssetAmount(
	chatID, tgUserID int64,
	BotMsgID int,
	msgText string,
	txData *t.TempTransactionData,
) error {
	// fmt.Println("raw", msgText)
	selectedPair := strings.TrimPrefix(msgText, "tx_pair_chosen_")

	// fmt.Println("after trim", selectedPair)

	_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(chatID, BotMsgID))

	result, err := s.transactionValidateInput(selectedPair, "pair")
	if err != nil {
		errorText := result.(string)
		msg := tgbotapi.NewMessage(chatID, errorText)
		msg.ParseMode = "Markdown"
		return s.sendTemporaryMessage(msg, tgUserID, 20*time.Second)
	}

	// fmt.Println("result", result)

	txData.Pair = result.(string)

	msg := tgbotapi.NewMessage(chatID,
		"Enter the asset amount (e.g. 1234, 12.34).")
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Cancel", "cancel_action"),
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

	result, err := s.transactionValidateInput(msgText, "amount")
	if err != nil {
		errorText := result.(string)
		msg := tgbotapi.NewMessage(chatID, errorText)
		msg.ParseMode = "Markdown"
		return s.sendTemporaryMessage(msg, tgUserID, 20*time.Second)
	}

	fmt.Printf("amountFloat: %s\n", reflect.TypeOf(result))

	txData.AssetAmount = result.(float64)

	msg := tgbotapi.NewMessage(chatID,
		"Enter the asset price (e.g. bought BTC for a 15500 usdt).")
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Cancel", "cancel_action"),
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

	result, err := s.transactionValidateInput(msgText, "price")
	if err != nil {
		errorText := result.(string)
		msg := tgbotapi.NewMessage(chatID, errorText)
		msg.ParseMode = "Markdown"
		return s.sendTemporaryMessage(msg, tgUserID, 20*time.Second)
	}

	fmt.Printf("amountFloat: %s\n", reflect.TypeOf(result))

	txData.AssetPrice = result.(float64)

	msg := tgbotapi.NewMessage(chatID,
		"Enter the transaction date in format *YYYY-MM-DD* (e.g. 2025-06-15)")
	msg.ParseMode = "Markdown"

	s.sessions.setState(tgUserID, "waiting_transaction_date")
	return s.sendTemporaryMessage(msg, tgUserID, 20*time.Second)
}

func (s *Service) transactionConfirmation(
	chatID, tgUserID int64,
	BotMsgID int,
	msgText string,
	txData *t.TempTransactionData,
) error {
	fmt.Println("date", msgText)

	_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(chatID, BotMsgID))

	result, err := s.transactionValidateInput(msgText, "date")
	if err != nil {
		errorText := result.(string)
		msg := tgbotapi.NewMessage(chatID, errorText)
		msg.ParseMode = "Markdown"
		return s.sendTemporaryMessage(msg, tgUserID, 20*time.Second)
	}

	fmt.Printf("amountFloat: %s\n", reflect.TypeOf(result))

	txData.TransactionDate = result.(time.Time)

	txData.USDAmount =
		txData.AssetAmount * txData.AssetPrice

	tableText := fmt.Sprintf(
		"*You are about to add a new transaction. Please confirm:*\n\n"+
			"```\n"+
			"| %-12s | %-13s | %-11s | %-10s | %-10s |\n"+
			"|--------------+---------------+-------------+------------+------------|\n"+
			"| %-12s | %-13.4f | %-11.4f | %-10.2f | %-10s |\n"+
			"```",
		"Pair", "Asset Amount", "Asset Price", "USD Amount", "Date",
		txData.Pair,
		txData.AssetAmount,
		txData.AssetPrice,
		txData.USDAmount,
		txData.TransactionDate.Format("2006-01-02"),
	)

	msg := tgbotapi.NewMessage(chatID, tableText)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Confirm", "tx_confirm_transaction"),
			tgbotapi.NewInlineKeyboardButtonData("Cancel", "cancel_action"),
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
			fmt.Sprintf("Transaction added successfully successfully: %s, %.2f USD!", txData.Pair, txData.USDAmount)),
		tgUserID,
		20*time.Second)
	if err != nil {
		return err
	}

	return s.showMainMenu(chatID, tgUserID)
}

func (s *Service) transactionValidateInput(rawText string, inputType string) (any, error) {
	text := strings.TrimSpace(rawText)

	switch inputType {
	case "pair":
		re := regexp.MustCompile(`[\s/\\]+`)
		cleaned := strings.ToUpper(
			re.ReplaceAllString(text, ""),
		)

		validRe := regexp.MustCompile(`^[A-Z]{4,12}$`) // ability to add USDT
		if !validRe.MatchString(cleaned) {
			return "Wrong data provided for *'pair'*. Only characters allowed (e.g. 'btc usdt', 'ETHUSDT'). Please try again.",
				fmt.Errorf("invalid data")
		}
		return cleaned, nil

	case "amount", "price":
		if !regexp.MustCompile(`^\d+(\.\d+)?$`).MatchString(text) {
			return fmt.Sprintf("Wrong data provided for *'%s'*. Only digits allowed (e.g. 1234, 12.34). Please try again.", inputType),
				fmt.Errorf("invalid data")
		}
		val, _ := strconv.ParseFloat(text, 64)
		return val, nil

	case "date":
		if !regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`).MatchString(text) {
			return "Wrong date format. Please use YYYY-MM-DD (e.g. 2024-06-14).", fmt.Errorf("invalid date format")
		}
		parsedTime, err := time.Parse("2006-01-02", text)
		if err != nil {
			return "Could not parse the date. Please try again", err
		}
		return parsedTime, nil

	default:
		return nil, fmt.Errorf("unknown input type")
	}
}

func (s *Service) mergeUniqueTxPairs(defaultPairs, topPairs []string) []string {
	unique := make(map[string]struct{})
	var result []string

	for _, pair := range append(defaultPairs, topPairs...) {
		// pair = strings.ToUpper(strings.TrimSpace(pair)) // на всякий случай
		if _, exists := unique[pair]; !exists {
			unique[pair] = struct{}{}
			result = append(result, pair)
		}
	}
	return result
}
