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
		"Please enter a currency pair (example: BTCUSDT, ETHUSDT etc. 'btc usdt' - also fine.)")
	msg.ParseMode = "Markdown"

	s.sessions.setState(tgUserID, "waiting_transaction_pair")
	return s.sendTemporaryMessage(msg, tgUserID, 20*time.Second)
}

func (s *Service) askTransactionAssetAmount(
	chatID, tgUserID int64,
	BotMsgID int,
	msgText string,
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

	pair := result.(string)

	session, _ := s.sessions.getSessionVars(tgUserID)
	session.TempTransaction.Pair = pair

	msg := tgbotapi.NewMessage(chatID,
		"Enter the asset amount (example: 1234, 12.34)")
	msg.ParseMode = "Markdown"

	s.sessions.setState(tgUserID, "waiting_transaction_asset_amount")
	return s.sendTemporaryMessage(msg, tgUserID, 20*time.Second)
}

func (s *Service) askTransactionAssetPrice(
	chatID, tgUserID int64,
	BotMsgID int,
	msgText string,
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

	amount := result.(float64)
	fmt.Printf("amountFloat: %s\n", reflect.TypeOf(amount))

	session, _ := s.sessions.getSessionVars(tgUserID)
	session.TempTransaction.AssetAmount = amount

	msg := tgbotapi.NewMessage(chatID,
		"Enter the asset price (for example, bought BTC for a 15500 usdt)")
	msg.ParseMode = "Markdown"

	s.sessions.setState(tgUserID, "waiting_transaction_asset_price")
	return s.sendTemporaryMessage(msg, tgUserID, 20*time.Second)
}

func (s *Service) askTransactionDate(
	chatID, tgUserID int64,
	BotMsgID int,
	msgText string,
) error {
	fmt.Println("price", msgText)
	// fmt.Printf("msgText: %s\n", reflect.TypeOf(msgText))

	_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(chatID, BotMsgID))

	// priceStr := strings.TrimSpace(msgText)
	// priceFloat, _ := strconv.ParseFloat(priceStr, 64)

	// fmt.Printf("amountFloat: %s\n", reflect.TypeOf(priceFloat))

	// session, _ := s.sessions.getSessionVars(tgUserID)
	// session.TempTransaction.AssetAmount = float64(priceFloat)

	result, err := s.transactionValidateInput(msgText, "amount")
	if err != nil {
		errorText := result.(string)
		msg := tgbotapi.NewMessage(chatID, errorText)
		msg.ParseMode = "Markdown"
		return s.sendTemporaryMessage(msg, tgUserID, 20*time.Second)
	}

	price := result.(float64)
	fmt.Printf("amountFloat: %s\n", reflect.TypeOf(price))

	session, _ := s.sessions.getSessionVars(tgUserID)
	session.TempTransaction.AssetAmount = price

	msg := tgbotapi.NewMessage(chatID,
		"Enter the asset price (for example, bought BTC for a 15500 usdt)")
	msg.ParseMode = "Markdown"

	s.sessions.setState(tgUserID, "waiting_transaction_asset_date")
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
			return "Wrong data provided for *'pair'*. Only characters allowed. Example: 'btc usdt', 'ETHUSDT'. Please try again",
				fmt.Errorf("invalid data")
		}
		return cleaned, nil

	case "amount", "price":
		if !regexp.MustCompile(`^\d+(\.\d+)?$`).MatchString(text) {
			return fmt.Sprintf("Wrong data provided for *'%s'*. Only digits allowed. Example: 1234, 12.34. Please try again", inputType),
				fmt.Errorf("invalid data")
		}
		val, _ := strconv.ParseFloat(text, 64)
		return val, nil

	default:
		return nil, fmt.Errorf("unknown input type")
	}
}
