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

// displays the full PnL report with current prices (like screenshot 1)
func (s *Service) showPortfolioAdvancedReport(ctx context.Context, chatID, tgUserID, dbUserID int64, BotMsgID int) error {
	_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(chatID, BotMsgID))

	// show loading message since this operation can take a few seconds
	loadingMsg := tgbotapi.NewMessage(chatID, "🔄 *Generating comprehensive PnL report...*\n\nFetching current prices and calculating metrics...")
	loadingMsg.ParseMode = "Markdown"
	loadingMessage, err := s.bot.Send(loadingMsg)
	if err != nil {
		log.Warn("Failed to send loading message", "error", err)
	}

	// get aggregated transaction data from database
	reportData, err := s.store.GetReportData(ctx, dbUserID)
	if err != nil {
		log.Error("Failed to get report data", "error", err, "user_id", dbUserID)

		// delete loading message if it was sent
		if loadingMessage.MessageID != 0 {
			_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(chatID, loadingMessage.MessageID))
		}

		return s.sendTemporaryMessage(
			tgbotapi.NewMessage(chatID, "❌ Sorry, couldn't retrieve your transaction data. Please try again."),
			tgUserID, 20*time.Second)
	}

	// check if user has any active positions
	if len(reportData) == 0 {
		// delete loading message
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

	// initialize PnL calculator with Binance API
	pnlCalc := &PnLCalculator{
		binanceAPIURL: s.cfg.BinanceAPIURL,
		httpClient:    &http.Client{Timeout: 15 * time.Second},
	}

	// calculate comprehensive PnL report
	report, err := s.calculateAdvancedReport(ctx, pnlCalc, reportData)
	if err != nil {
		log.Error("Failed to calculate PnL report", "error", err, "user_id", dbUserID)

		// delete loading message
		if loadingMessage.MessageID != 0 {
			_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(chatID, loadingMessage.MessageID))
		}

		errorMsg := "❌ Failed to fetch current prices or calculate PnL.\n\n"

		// provide specific error messages based on the error type
		errorStr := err.Error()
		if strings.Contains(errorStr, "no valid prices found") {
			errorMsg += "🔍 **Price Data Issue:**\n"
			errorMsg += "None of your cryptocurrency pairs were found on Binance.\n\n"
			errorMsg += "**Possible reasons:**\n"
			errorMsg += "• Pairs might not be listed on Binance\n"
			errorMsg += "• Incorrect pair format (should be like BTCUSDT)\n"
			errorMsg += "• Pairs might have been delisted\n\n"
			errorMsg += "💡 **Tip:** Check if your pairs are actively traded on Binance."
		} else if strings.Contains(errorStr, "no valid price data available") {
			errorMsg += "🔍 **Price Data Issue:**\n"
			errorMsg += "No current price data available for your pairs.\n\n"
			errorMsg += "This might be due to:\n"
			errorMsg += "• Binance API maintenance\n"
			errorMsg += "• Network connectivity issues\n"
			errorMsg += "• Temporary API unavailability\n\n"
			errorMsg += "Please try again in a few minutes."
		} else if strings.Contains(errorStr, "Binance API error") {
			errorMsg += "🔌 **Binance API Error:**\n"
			errorMsg += fmt.Sprintf("API returned an error: %s\n\n", err.Error())
			errorMsg += "This might be due to:\n"
			errorMsg += "• API rate limiting (too many requests)\n"
			errorMsg += "• Binance server issues\n"
			errorMsg += "• API maintenance\n\n"
			errorMsg += "Please wait a few minutes and try again."
		} else {
			errorMsg += "This might be due to:\n"
			errorMsg += "• Network connectivity issues\n"
			errorMsg += "• Binance API temporary unavailability\n"
			errorMsg += "• Invalid currency pairs\n\n"
			errorMsg += "Please try again in a few minutes."
		}

		return s.sendTemporaryMessage(
			tgbotapi.NewMessage(chatID, errorMsg),
			tgUserID, 30*time.Second)
	}

	// format the report for display
	reportText := s.formatAdvancedReport(report)

	// delete loading message
	if loadingMessage.MessageID != 0 {
		_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(chatID, loadingMessage.MessageID))
	}

	// send the comprehensive report
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

// performs all PnL calculations using the mathematical formulas
func (s *Service) calculateAdvancedReport(ctx context.Context, calc *PnLCalculator, reportData []t.CurrencyPnLData) (*t.GeneralReport, error) {
	if len(reportData) == 0 {
		return &t.GeneralReport{
			CurrencyData: []t.CurrencyPnLData{},
			LastUpdated:  time.Now().Format("2006-01-02 15:04:05"),
		}, nil
	}

	// extract all assets and convert to pairs for API call
	pairs := make([]string, len(reportData))
	for i, data := range reportData {
		pairs[i] = data.Asset + "USDT" //FIXME move to global vars (or config)
	}

	// FIXME here we have logic where we pop pairs that have no dresponse from Binance,
	// but we make 1 api request with list of pairs
	// we should make 1 api request per pair, and then pop pairs that have no response (new branch)

	// fetch current prices
	currentPrices, err := calc.FetchCurrentPrices(ctx, pairs)
	if err != nil {
		return nil, fmt.Errorf("fetch current prices: %w", err)
	}

	// calculate PnL for each currency pair
	var calculatedData []t.CurrencyPnLData
	var totalInvested, totalCurrentValue float64
	var skippedPairs []string

	for _, data := range reportData {
		pair := data.Asset + "USDT" // FIXME Convert asset ticker to USDT pair for price lookup (check if its required)
		currentPrice, priceExists := currentPrices[pair]
		if !priceExists {
			log.Warn("No current price found for asset", "asset", data.Asset, "pair", pair)
			skippedPairs = append(skippedPairs, data.Asset)
			continue
		}

		// apply the mathematical formulas:
		data.CurrentPrice = currentPrice
		data.CurrentValueUSD = data.TotalAssetAmount * currentPrice
		data.PnLUSD = data.CurrentValueUSD - data.TotalInvestedUSD

		// calculate PnL percentage - handle negative assets properly
		if data.TotalInvestedUSD > 0 {
			data.PnLPercentage = ((data.CurrentValueUSD / data.TotalInvestedUSD) - 1) * 100
		} else if data.TotalInvestedUSD < 0 {
			// negative invested means they took out more than they put in
			// in this case, any remaining value is pure profit
			data.PnLPercentage = 999.99 // Indicates "pure profit" scenario
		} else {
			// edge case: exactly zero net invested
			data.PnLPercentage = 0
		}

		data.LastUpdated = time.Now().Format("2006-01-02 15:04:05")

		calculatedData = append(calculatedData, data)
		totalInvested += data.TotalInvestedUSD
		totalCurrentValue += data.CurrentValueUSD
	}

	// log skipped pairs if any
	if len(skippedPairs) > 0 {
		log.Warn("Skipped pairs due to missing price data", "skipped_pairs", skippedPairs, "skipped_count", len(skippedPairs))
	}

	// check if we have any valid data to report
	if len(calculatedData) == 0 {
		return nil, fmt.Errorf("no valid price data available for any of the requested pairs")
	}

	// calculate overall portfolio metrics
	totalPnL := totalCurrentValue - totalInvested
	var totalPnLPercent float64
	if totalInvested > 0 {
		totalPnLPercent = ((totalCurrentValue / totalInvested) - 1) * 100
	} else if totalInvested < 0 {
		totalPnLPercent = 999.99
	}

	report := &t.GeneralReport{
		CurrencyData:       calculatedData,
		TotalInvestedUSD:   totalInvested,
		TotalCurrentUSD:    totalCurrentValue,
		TotalPnLUSD:        totalPnL,
		TotalPnLPercentage: totalPnLPercent,
		LastUpdated:        time.Now().Format("2006-01-02 15:04:05"),
	}

	// add skipped pairs information to the report for display
	if len(skippedPairs) > 0 {
		// store skipped pairs in the report for later display
		// we'll add this as a custom field or handle it in the format function
		log.Info("Report generated with some skipped pairs", "processed_pairs", len(calculatedData), "skipped_pairs", len(skippedPairs))
	}

	return report, nil
}

// creates the advanced report with the specific format requested:
// asset | total_asset_amount | total_invested_amount_usd | PnL% | PnL USD | current_value | average_purchase_price
func (s *Service) formatAdvancedReport(report *t.GeneralReport) string {
	if len(report.CurrencyData) == 0 {
		return "*📊 General Portfolio Report*\n\n" +
			"🤷‍♂️ No active positions found.\n" +
			"Add some transactions to see your PnL analysis!"
	}

	var builder strings.Builder

	// header
	builder.WriteString("📊 *Advanced Portfolios Report*\n")
	builder.WriteString(fmt.Sprintf("📅 Generated: `%s`\n\n", report.LastUpdated))

	// individual currency data
	builder.WriteString("💰 *Assets over all portfolios:*\n\n")

	for i, data := range report.CurrencyData {
		// choose emoji based on PnL
		var pnlEmoji string
		switch {
		case data.PnLUSD > 0:
			pnlEmoji = "🟢"
		case data.PnLUSD < 0:
			pnlEmoji = "🔴"
		default:
			pnlEmoji = "⚪"
		}

		// use the asset ticker directly (we already have BTC, ETH, etc.)
		baseCurrency := data.Asset

		// format invested amount - handle negative case (when more was taken out than invested)
		var investedText string
		if data.TotalInvestedUSD >= 0 {
			investedText = fmt.Sprintf("Net Invested: `$%.2f`", data.TotalInvestedUSD)
		} else {
			// negative invested means they took out more than they put in
			investedText = fmt.Sprintf("Net Profit Taken: `$%.2f`", -data.TotalInvestedUSD)
		}

		// format PnL with proper signs
		var pnlUSDText, pnlPercentText string
		if data.PnLUSD >= 0 {
			pnlUSDText = fmt.Sprintf("+$%.2f", data.PnLUSD)
		} else {
			pnlUSDText = fmt.Sprintf("$%.2f", data.PnLUSD) // negative sign already included
		}

		if data.PnLPercentage >= 0 {
			pnlPercentText = fmt.Sprintf("+%.2f%%", data.PnLPercentage)
		} else {
			pnlPercentText = fmt.Sprintf("%.2f%%", data.PnLPercentage) // already has negative sign
		}

		// handle special "pure profit" case
		if data.PnLPercentage == 999.99 {
			pnlPercentText = "🚀 PURE PROFIT"
		}

		// show break-even status
		var breakEvenStatus string
		if data.CurrentPrice < data.AveragePurchasePrice {
			breakEvenStatus = "📉 *Below break-even*"
		} else {
			breakEvenStatus = "📈 *Above break-even*"
		}

		builder.WriteString(fmt.Sprintf(
			"%s *%s*\n"+
				"Holdings: `%.8g %s`\n"+
				"%s\n"+
				"Current Value: `$%.2f` @ `$%.2f`\n"+
				"Avg Buy Price: `$%.2f` %s\n"+
				"PnL: `%s` (`%s`)\n",
			pnlEmoji,
			data.Asset,
			data.TotalAssetAmount,
			baseCurrency,
			investedText,
			data.CurrentValueUSD,
			data.CurrentPrice,
			data.AveragePurchasePrice,
			breakEvenStatus,
			pnlUSDText,
			pnlPercentText,
		))

		// add separator except for the last item
		if i < len(report.CurrencyData)-1 {
			builder.WriteString("\n" + strings.Repeat("─", 19) + "\n\n")
		}
	}

	// overall portfolio summary
	builder.WriteString("\n" + strings.Repeat("—", 20) + "\n\n")

	var totalEmoji string
	switch {
	case report.TotalPnLUSD > 0:
		totalEmoji = "🚀"
	case report.TotalPnLUSD < 0:
		totalEmoji = "📉"
	default:
		totalEmoji = "⚖️"
	}

	// format total amounts with proper signs
	var totalInvestedText string
	if report.TotalInvestedUSD >= 0 {
		totalInvestedText = fmt.Sprintf("💸 Net Invested: `$%.2f`", report.TotalInvestedUSD)
	} else {
		totalInvestedText = fmt.Sprintf("💰 Net Profit Taken: `$%.2f`", -report.TotalInvestedUSD)
	}

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

	// handle special "pure profit" case for total
	if report.TotalPnLPercentage == 999.99 {
		totalPnLPercentText = "🚀 PURE PROFIT"
	}

	builder.WriteString(fmt.Sprintf(
		"%s *Total Overview:*\n\n"+
			"%s\n"+
			"💎 Current Value: `$%.2f`\n"+
			"📊 Total PnL: `%s` (`%s`)\n",
		totalEmoji,
		totalInvestedText,
		report.TotalCurrentUSD,
		totalPnLUSDText,
		totalPnLPercentText,
	))

	return builder.String()
}
