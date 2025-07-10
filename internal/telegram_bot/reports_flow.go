package telegram_bot

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.com/avolkov/wood_post/pkg/log"
	t "gitlab.com/avolkov/wood_post/pkg/types"
)

func (s *Service) gfReportsMain(chatID, tgUserID int64, BotMsgID int) error {
	_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(chatID, BotMsgID))

	actions := []t.Actiontype{
		{TgText: "General (historical cost basis)", CallBackName: "gf_reports_general"},
		{TgText: "Advanced (PnL)", CallBackName: "gf_reports_advanced"},
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

// FetchCurrentPrices fetches current prices for specific cryptocurrency pairs from Binance API
// Makes a single request with the pairs array to get only the prices we need
func (calc *PnLCalculator) FetchCurrentPrices(ctx context.Context, pairs []string) (map[string]float64, error) {
	if len(pairs) == 0 {
		return make(map[string]float64), nil
	}

	log.Info("Fetching current prices from Binance API", "pairs_count", len(pairs), "pairs", pairs)

	// Prepare the symbols array parameter for Binance API
	// Format: ["BTCUSDT","ETHUSDT","BNBUSDT"]
	symbolsJSON, err := json.Marshal(pairs)
	if err != nil {
		return nil, fmt.Errorf("marshal pairs to JSON: %w", err)
	}

	// Build API URL with symbols parameter
	apiURL := "https://api.binance.com/api/v3/ticker/price?symbols=" + string(symbolsJSON)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create HTTP request: %w", err)
	}

	// Add headers to identify the request
	// FIXME why is it here?
	// req.Header.Set("User-Agent", "wood_post_bot/1.0")

	log.Info("Making request to Binance API", "url", apiURL)

	resp, err := calc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute HTTP request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// Try to parse as Binance error response
		var binanceErr t.BinanceErrorResponse
		if err := json.Unmarshal(body, &binanceErr); err == nil {
			return nil, fmt.Errorf("Binance API error (code %d): %s", binanceErr.Code, binanceErr.Msg)
		}
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the response - should be an array of price data for our specific pairs
	var priceResponses []t.BinancePriceResponse
	if err := json.Unmarshal(body, &priceResponses); err != nil {
		return nil, fmt.Errorf("unmarshal price response: %w", err)
	}

	log.Info("Received price data from Binance", "received_symbols", len(priceResponses))

	// Create a map of prices
	priceMap := make(map[string]float64)
	for _, priceResp := range priceResponses {
		price, err := strconv.ParseFloat(priceResp.Price, 64)
		if err != nil {
			log.Warn("Failed to parse price", "symbol", priceResp.Symbol, "price", priceResp.Price, "error", err)
			continue
		}
		priceMap[priceResp.Symbol] = price
		log.Info("Parsed price for pair", "pair", priceResp.Symbol, "price", price)
	}

	// Check which pairs were missing from the response
	var missingPairs []string
	for _, pair := range pairs {
		if _, exists := priceMap[pair]; !exists {
			missingPairs = append(missingPairs, pair)
			log.Warn("Price not found for pair", "pair", pair)
		}
	}

	// Log results
	log.Info("Price fetching completed",
		"requested_pairs", len(pairs),
		"found_pairs", len(priceMap),
		"missing_pairs", len(missingPairs))

	if len(missingPairs) > 0 {
		log.Warn("Some pairs were not found on Binance", "missing_pairs", missingPairs)
	}

	// Return error if no prices were found at all
	if len(priceMap) == 0 {
		return nil, fmt.Errorf("no valid prices found for any of the requested pairs: %v", pairs)
	}

	return priceMap, nil
}
