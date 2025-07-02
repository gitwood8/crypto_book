package telegram_bot

import (
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	t "gitlab.com/avolkov/wood_post/pkg/types"
)

func (s *Service) gfReportsMain(chatID, tgUserID int64, BotMsgID int) error {
	_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(chatID, BotMsgID))

	actions := []t.Actiontype{
		{TgText: "General report", CallBackName: "gf_reports_general"},
		{TgText: "Advanced report", CallBackName: "gf_reports_advanced"},
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
