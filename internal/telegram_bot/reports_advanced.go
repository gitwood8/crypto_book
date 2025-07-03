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

// showPortfolioAdvancedReport displays the full PnL report with current prices (like screenshot 1)
func (s *Service) showPortfolioAdvancedReport(ctx context.Context, chatID, tgUserID, dbUserID int64, BotMsgID int) error {
	_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(chatID, BotMsgID))

	// Show loading message since this operation can take a few seconds
	loadingMsg := tgbotapi.NewMessage(chatID, "🔄 *Generating comprehensive PnL report...*\n\nFetching current prices and calculating metrics...")
	loadingMsg.ParseMode = "Markdown"
	loadingMessage, err := s.bot.Send(loadingMsg)
	if err != nil {
		log.Warn("Failed to send loading message", "error", err)
	}

	// Get aggregated transaction data from database
	reportData, err := s.store.GetReportData(ctx, dbUserID)
	if err != nil {
		log.Error("Failed to get report data", "error", err, "user_id", dbUserID)

		// Delete loading message if it was sent
		if loadingMessage.MessageID != 0 {
			_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(chatID, loadingMessage.MessageID))
		}

		return s.sendTemporaryMessage(
			tgbotapi.NewMessage(chatID, "❌ Sorry, couldn't retrieve your transaction data. Please try again."),
			tgUserID, 20*time.Second)
	}

	// Check if user has any active positions
	if len(reportData) == 0 {
		// Delete loading message
		if loadingMessage.MessageID != 0 {
			_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(chatID, loadingMessage.MessageID))
		}

		msg := tgbotapi.NewMessage(chatID, "📊 *Advanced PnL Report*\n\n🤷‍♂️ No active positions found.\n\nYou need to have transactions to generate a PnL report. Start by adding some BUY transactions!")
		msg.ParseMode = "Markdown"
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Add Transaction", "gf_add_transaction"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Main Menu", "cancel_action"),
			),
		)
		return s.sendTemporaryMessage(msg, tgUserID, 30*time.Second)
	}

	// Initialize PnL calculator with Binance API
	pnlCalc := &PnLCalculator{
		binanceAPIURL: s.cfg.BinanceAPIURL,
		httpClient:    &http.Client{Timeout: 15 * time.Second},
	}

	// Calculate comprehensive PnL report
	report, err := s.calculateAdvancedReport(ctx, pnlCalc, reportData)
	if err != nil {
		log.Error("Failed to calculate PnL report", "error", err, "user_id", dbUserID)

		// Delete loading message
		if loadingMessage.MessageID != 0 {
			_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(chatID, loadingMessage.MessageID))
		}

		errorMsg := "❌ Failed to fetch current prices or calculate PnL.\n\n"
		errorMsg += "This might be due to:\n"
		errorMsg += "• Network connectivity issues\n"
		errorMsg += "• Binance API problems\n"
		errorMsg += "• Invalid currency pairs\n\n"
		errorMsg += "Please try again in a few minutes."

		return s.sendTemporaryMessage(
			tgbotapi.NewMessage(chatID, errorMsg),
			tgUserID, 30*time.Second)
	}

	// Format the report for display
	reportText := s.formatAdvancedReport(report)

	// Delete loading message
	if loadingMessage.MessageID != 0 {
		_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(chatID, loadingMessage.MessageID))
	}

	// Send the comprehensive report
	msg := tgbotapi.NewMessage(chatID, reportText)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 General Report", "gf_reports_general"),
			tgbotapi.NewInlineKeyboardButtonData("➕ Add Transaction", "gf_add_transaction"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Main Menu", "cancel_action"),
		),
	)

	log.Info("Advanced PnL report sent successfully", "user_id", dbUserID)
	return s.sendTemporaryMessage(msg, tgUserID, 120*time.Second)
}

// Helper methods for advanced report calculations
func (s *Service) calculateAdvancedReport(ctx context.Context, calc *PnLCalculator, reportData []t.CurrencyPnLData) (*t.GeneralReport, error) {
	if len(reportData) == 0 {
		return &t.GeneralReport{
			CurrencyData: []t.CurrencyPnLData{},
			LastUpdated:  time.Now().Format("2006-01-02 15:04:05"),
		}, nil
	}

	// Extract all pairs for API call
	pairs := make([]string, len(reportData))
	for i, data := range reportData {
		pairs[i] = data.Pair
	}

	// Fetch current prices
	currentPrices, err := calc.FetchCurrentPrices(ctx, pairs)
	if err != nil {
		return nil, fmt.Errorf("fetch current prices: %w", err)
	}

	// Calculate PnL for each currency pair
	var calculatedData []t.CurrencyPnLData
	var totalInvested, totalCurrentValue float64

	for _, data := range reportData {
		currentPrice, priceExists := currentPrices[data.Pair]
		if !priceExists {
			log.Warn("No current price found for pair", "pair", data.Pair)
			continue
		}

		// Apply the mathematical formulas:
		data.CurrentPrice = currentPrice
		data.CurrentValueUSD = data.TotalAssetAmount * currentPrice
		data.PnLUSD = data.CurrentValueUSD - data.TotalInvestedUSD

		// Calculate PnL percentage
		if data.TotalInvestedUSD > 0 {
			data.PnLPercentage = ((data.CurrentValueUSD / data.TotalInvestedUSD) - 1) * 100
		} else if data.TotalInvestedUSD < 0 {
			data.PnLPercentage = 999.99 // Indicates "pure profit" scenario
		} else {
			data.PnLPercentage = 0
		}

		data.LastUpdated = time.Now().Format("2006-01-02 15:04:05")

		calculatedData = append(calculatedData, data)
		totalInvested += data.TotalInvestedUSD
		totalCurrentValue += data.CurrentValueUSD
	}

	// Calculate overall portfolio metrics
	totalPnL := totalCurrentValue - totalInvested
	var totalPnLPercent float64
	if totalInvested > 0 {
		totalPnLPercent = ((totalCurrentValue / totalInvested) - 1) * 100
	} else if totalInvested < 0 {
		totalPnLPercent = 999.99
	}

	return &t.GeneralReport{
		CurrencyData:       calculatedData,
		TotalInvestedUSD:   totalInvested,
		TotalCurrentUSD:    totalCurrentValue,
		TotalPnLUSD:        totalPnL,
		TotalPnLPercentage: totalPnLPercent,
		LastUpdated:        time.Now().Format("2006-01-02 15:04:05"),
	}, nil
}

func (s *Service) formatAdvancedReport(report *t.GeneralReport) string {
	if len(report.CurrencyData) == 0 {
		return "*📊 General Portfolio Report*\n\n" +
			"🤷‍♂️ No active positions found.\n" +
			"Add some transactions to see your PnL analysis!"
	}

	var builder strings.Builder

	// Header
	builder.WriteString("📊 *General Portfolio Report*\n")
	builder.WriteString(fmt.Sprintf("📅 Generated: `%s`\n\n", report.LastUpdated))

	// Individual currency data
	builder.WriteString("💰 *Individual Assets:*\n\n")

	for i, data := range report.CurrencyData {
		// Choose emoji based on PnL
		var pnlEmoji string
		switch {
		case data.PnLUSD > 0:
			pnlEmoji = "🟢"
		case data.PnLUSD < 0:
			pnlEmoji = "🔴"
		default:
			pnlEmoji = "⚪"
		}

		baseCurrency := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(data.Pair, "USDT"), "USDC"), "USD"), "EUR")

		// Format PnL with proper signs
		var pnlUSDText, pnlPercentText string
		if data.PnLUSD >= 0 {
			pnlUSDText = fmt.Sprintf("+$%.2f", data.PnLUSD)
		} else {
			pnlUSDText = fmt.Sprintf("$%.2f", data.PnLUSD)
		}

		if data.PnLPercentage >= 0 {
			pnlPercentText = fmt.Sprintf("+%.2f%%", data.PnLPercentage)
		} else {
			pnlPercentText = fmt.Sprintf("%.2f%%", data.PnLPercentage)
		}

		if data.PnLPercentage == 999.99 {
			pnlPercentText = "🚀 PURE PROFIT"
		}

		builder.WriteString(fmt.Sprintf(
			"%s *%s*\n"+
				"Holdings: `%.8g %s`\n"+
				"Net Invested: `$%.2f`\n"+
				"Current Value: `$%.2f` @ `$%.2f`\n"+
				"Avg Buy Price: `$%.2f`\n"+
				"PnL: `%s` (`%s`)\n",
			pnlEmoji,
			data.Pair,
			data.TotalAssetAmount,
			baseCurrency,
			data.TotalInvestedUSD,
			data.CurrentValueUSD,
			data.CurrentPrice,
			data.AveragePurchasePrice,
			pnlUSDText,
			pnlPercentText,
		))

		// Add separator except for the last item
		if i < len(report.CurrencyData)-1 {
			builder.WriteString("\n" + strings.Repeat("-", 12) + "\n\n")
		}
	}

	// Overall portfolio summary
	builder.WriteString("\n" + strings.Repeat("═", 30) + "\n\n")
	builder.WriteString("📈 *Portfolio Summary:*\n\n")

	var totalEmoji string
	switch {
	case report.TotalPnLUSD > 0:
		totalEmoji = "🚀"
	case report.TotalPnLUSD < 0:
		totalEmoji = "📉"
	default:
		totalEmoji = "⚖️"
	}

	// Format total PnL
	var totalPnLUSDText, totalPnLPercentText string
	if report.TotalPnLUSD >= 0 {
		totalPnLUSDText = fmt.Sprintf("+$%.2f", report.TotalPnLUSD)
	} else {
		totalPnLUSDText = fmt.Sprintf("$%.2f", report.TotalPnLUSD)
	}

	if report.TotalPnLPercentage >= 0 {
		totalPnLPercentText = fmt.Sprintf("+%.2f%%", report.TotalPnLPercentage)
	} else {
		totalPnLPercentText = fmt.Sprintf("%.2f%%", report.TotalPnLPercentage)
	}

	if report.TotalPnLPercentage == 999.99 {
		totalPnLPercentText = "🚀 PURE PROFIT"
	}

	builder.WriteString(fmt.Sprintf(
		"%s *Total Overview*\n"+
			"💸 Net Invested: `$%.2f`\n"+
			"💎 Current Value: `$%.2f`\n"+
			"📊 Total PnL: `%s` (`%s`)\n",
		totalEmoji,
		report.TotalInvestedUSD,
		report.TotalCurrentUSD,
		totalPnLUSDText,
		totalPnLPercentText,
	))

	return builder.String()
}
