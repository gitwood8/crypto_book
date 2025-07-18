package telegram_bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	t "gitlab.com/avolkov/wood_post/pkg/types"
)

func (s *Service) gfTransactionsDelete(ctx context.Context, chatID, tgUserID, dbUserID int64, BotMsgID int) error {
	_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(chatID, BotMsgID))

	var tx []t.Transaction

	tx, err := s.store.GetLast5TransactionsForUser(ctx, dbUserID)
	if err != nil {
		return fmt.Errorf("get last 5 transactions: %w", err)
	}

	if len(tx) == 0 {
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

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, t := range tx {
		var typeEmoji string
		switch strings.ToLower(t.Type) {
		case "buy":
			typeEmoji = "🟢"
		case "sell":
			typeEmoji = "🔴"
		default:
			typeEmoji = "🔵"
		}

		// FIXME: add tx id to callback and use it in gfDeleteTransactionConfirmed
		txText := fmt.Sprintf("%s%s | %.8g %s | %.2f usd", typeEmoji, t.Type, t.AssetAmount, t.Asset, t.USDAmount)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(txText, "gf_delete_transaction_confirmation_"+strconv.FormatInt(t.ID, 10)),
		))
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Back", "cancel_action"),
	))

	msg := tgbotapi.NewMessage(chatID, "Select a transaction that you want to delete:")
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)

	return s.sendTemporaryMessage(msg, tgUserID, 30*time.Second)
}

func (s *Service) gfDeleteTransactionConfirmation(
	chatID, tgUserID int64,
	BotMsgID int,
	cbData string,
	txData *t.TempTransactionData,
) error {
	_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(chatID, BotMsgID))

	txID := strings.TrimPrefix(cbData, "gf_delete_transaction_confirmation_")
	txIDInt, err := strconv.ParseInt(txID, 10, 64)
	if err != nil {
		return fmt.Errorf("parse tx id: %w", err)
	}

	txData.ID = txIDInt

	msg := tgbotapi.NewMessage(chatID, "Are you sure you want to delete this transaction?")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Yes, delete", "gf_delete_transaction_confirmed"),
			tgbotapi.NewInlineKeyboardButtonData("Back", "gf_delete_transaction"),
		),
	)

	return s.sendTemporaryMessage(msg, tgUserID, 20*time.Second)
}

func (s *Service) gfDeleteTransactionConfirmed(
	ctx context.Context,
	chatID, tgUserID, dbUserID, txID int64,
	BotMsgID int,
) error {
	err := s.store.DeleteTransaction(ctx, dbUserID, txID)
	if err != nil {
		return fmt.Errorf("delete transaction: %w", err)
	}
	err = s.editMessageText(
		chatID,
		BotMsgID,
		"Transaction deleted successfully.")
	if err != nil {
		return err
	}
	return s.showMainMenu(chatID, tgUserID)
}
