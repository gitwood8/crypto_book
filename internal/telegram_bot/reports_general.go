package telegram_bot

import (
	"context"
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.com/avolkov/wood_post/pkg/log"
)

// showPortfolioGeneralReport displays the historical cost basis report (like screenshot 2)
func (s *Service) showPortfolioGeneralReport(ctx context.Context, chatID, tgUserID, dbUserID int64, BotMsgID int) error {
	_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(chatID, BotMsgID))

	// Get portfolio summaries for historical cost basis
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

	// Build the basic report message (like screenshot 2)
	var reportText strings.Builder
	reportText.WriteString("📊 *GENERAL PORTFOLIO REPORT*\n")
	reportText.WriteString("_(Historical cost basis only)_\n\n")

	var grandTotalUSD float64
	for i, summary := range summaries {
		if i > 0 {
			reportText.WriteString("\n")
		}

		reportText.WriteString(fmt.Sprintf("*Portfolio: %s*\n", summary.Name))

		var portfolioTotalUSD float64
		for _, asset := range summary.Assets {
			// Extract base asset from pair
			baseCurrency := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(asset.Pair, "USDT"), "USDC"), "USD"), "EUR")

			reportText.WriteString(fmt.Sprintf("%s: %.6g %s, invested: %.2f USD\n",
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

	reportText.WriteString(fmt.Sprintf("\n🎯 *GRAND TOTAL: %.2f USD*\n", grandTotalUSD))
	reportText.WriteString("\n💡 _This shows historical cost basis. For current PnL analysis, use the Advanced Report._")

	msg := tgbotapi.NewMessage(chatID, reportText.String())
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📈 Advanced PnL Report", "gf_reports_advanced"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Back to Reports", "gf_reports_main"),
			tgbotapi.NewInlineKeyboardButtonData("Main Menu", "cancel_action"),
		),
	)

	return s.sendTemporaryMessage(msg, tgUserID, 60*time.Second)
}
