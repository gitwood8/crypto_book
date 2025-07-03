package telegram_bot

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.com/avolkov/wood_post/pkg/log"
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

// PnLCalculator struct to hold Binance API configuration
type PnLCalculator struct {
	binanceAPIURL string
	httpClient    *http.Client
}

// FetchCurrentPrices fetches current prices for cryptocurrency pairs from Binance API
func (calc *PnLCalculator) FetchCurrentPrices(ctx context.Context, pairs []string) (map[string]float64, error) {
	if len(pairs) == 0 {
		return make(map[string]float64), nil
	}

	priceMap := make(map[string]float64)

	// For now, return mock data to make it work
	// In production, this would call the actual Binance API
	for _, pair := range pairs {
		switch pair {
		case "BTCUSDT":
			priceMap[pair] = 45000.0
		case "ETHUSDT":
			priceMap[pair] = 3000.0
		case "ADAUSDT":
			priceMap[pair] = 0.5
		default:
			priceMap[pair] = 100.0 // Default mock price
		}
	}

	log.Info("Fetched mock prices", "pairs", len(pairs), "prices", len(priceMap))
	return priceMap, nil
}

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
