package telegram_bot

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (s *Service) gfTransactionsMain(chatID, tgUserID int64, BotMsgID int) error {
	_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(chatID, BotMsgID))

	actions := []Action{
		{"Add transaction", "gf_add_transaction"},
		// {"Delete portfolio", "gf_portfolios_delete"},
		// {"Get default", "gf_portfolio_get_default"},
		// {"Change default", "gf_portfolio_change_default"},
		// {"Rename", "gf_portfolio_rename"},
		{"Back to main menu", "cancel_action"},
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

func (s *Service) askTransactionPair(chatID, tgUserID int64, BotMsgID int) error {
	_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(chatID, BotMsgID))
	msg := tgbotapi.NewMessage(chatID,
		"Please enter a currency pair (example: BTCUSDT, ETHUSDT etc. 'btc usdt' - also fine).")
	msg.ParseMode = "Markdown"

	s.sessions.setState(tgUserID, "waiting_transaction_pair")
	return s.sendTemporaryMessage(msg, tgUserID, 20*time.Second)
}

func (s *Service) askTransactionAssetAmount(
	chatID, tgUserID int64,
	BotMsgID int,
	msgText string,
	session *UserSession,
) error {
	fmt.Println("pair", msgText)
	_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(chatID, BotMsgID))

	result, err := s.transactionValidateInput(msgText, "pair")
	if err != nil {
		errorText := result.(string)
		msg := tgbotapi.NewMessage(chatID, errorText)
		msg.ParseMode = "Markdown"
		return s.sendTemporaryMessage(msg, tgUserID, 20*time.Second)
	}

	session.TempTransaction.Pair = result.(string)

	msg := tgbotapi.NewMessage(chatID,
		"Enter the asset amount (example: 1234, 12.34).")
	msg.ParseMode = "Markdown"

	s.sessions.setState(tgUserID, "waiting_transaction_asset_amount")
	return s.sendTemporaryMessage(msg, tgUserID, 20*time.Second)
}

func (s *Service) askTransactionAssetPrice(
	chatID, tgUserID int64,
	BotMsgID int,
	msgText string,
	session *UserSession,
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

	session.TempTransaction.AssetAmount = result.(float64)

	msg := tgbotapi.NewMessage(chatID,
		"Enter the asset price (e.g. bought BTC for a 15500 usdt).")
	msg.ParseMode = "Markdown"

	s.sessions.setState(tgUserID, "waiting_transaction_asset_price")
	return s.sendTemporaryMessage(msg, tgUserID, 20*time.Second)
}

func (s *Service) askTransactionDate(
	chatID, tgUserID int64,
	BotMsgID int,
	msgText string,
	session *UserSession,
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

	session.TempTransaction.AssetPrice = result.(float64)

	msg := tgbotapi.NewMessage(chatID,
		"Enter the transaction date in format *YYYY-MM-DD* (example: 2025-06-15)")
	msg.ParseMode = "Markdown"

	s.sessions.setState(tgUserID, "waiting_transaction_date")
	return s.sendTemporaryMessage(msg, tgUserID, 20*time.Second)
}

func (s *Service) TransactionConfirmation(
	chatID, tgUserID int64,
	BotMsgID int,
	msgText string,
	session *UserSession,
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

	session.TempTransaction.TransactionDate = result.(time.Time)

	session.TempTransaction.USDAmount =
		session.TempTransaction.AssetAmount * session.TempTransaction.AssetPrice

	tableText := fmt.Sprintf(
		"*You are about to add a new transaction. Please confirm:*\n\n"+
			"```\n"+
			"| %-12s | %-13s | %-11s | %-10s | %-10s |\n"+
			"|--------------+---------------+-------------+------------+------------|\n"+
			"| %-12s | %-13.4f | %-11.4f | %-10.2f | %-10s |\n"+
			"```",
		"Pair", "Asset Amount", "Asset Price", "USD Amount", "Date",
		session.TempTransaction.Pair,
		session.TempTransaction.AssetAmount,
		session.TempTransaction.AssetPrice,
		session.TempTransaction.USDAmount,
		session.TempTransaction.TransactionDate.Format("2006-01-02"),
	)

	msg := tgbotapi.NewMessage(chatID, tableText)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Confirm", "confirm_transaction"),
			tgbotapi.NewInlineKeyboardButtonData("Cancel", "cancel_action"),
		),
	)
	msg.ParseMode = "Markdown"

	s.sessions.setState(tgUserID, "waiting_transaction_confirmation")
	return s.sendTemporaryMessage(msg, tgUserID, 20*time.Second)
}

func (s *Service) transactionValidateInput(rawText string, inputType string) (any, error) {
	text := strings.TrimSpace(rawText)

	switch inputType {
	case "pair":
		re := regexp.MustCompile(`[\s/\\]+`)
		cleaned := strings.ToUpper(
			re.ReplaceAllString(text, ""),
		)

		validRe := regexp.MustCompile(`^[A-Z]{5,12}$`)
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
