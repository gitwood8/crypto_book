package types

// CurrencyPnLData represents PnL data for a specific currency pair across all portfolios
type CurrencyPnLData struct {
	Pair                 string  // "BTCUSDT"
	TotalAssetAmount     float64 // Total BTC held (sum of all BUY - SELL)
	TotalInvestedUSD     float64 // Total USD invested (sum of all BUY transactions)
	CurrentPrice         float64 // Current price from Binance API
	CurrentValueUSD      float64 // TotalAssetAmount * CurrentPrice
	PnLUSD               float64 // CurrentValueUSD - TotalInvestedUSD
	PnLPercentage        float64 // ((CurrentValueUSD / TotalInvestedUSD) - 1) * 100
	AveragePurchasePrice float64 // TotalInvestedUSD / TotalAssetAmount
	LastUpdated          string  // Current date
}

// GeneralReport represents the complete general report data
type GeneralReport struct {
	CurrencyData       []CurrencyPnLData
	TotalInvestedUSD   float64 // Sum of all investments
	TotalCurrentUSD    float64 // Sum of all current values
	TotalPnLUSD        float64 // Total profit/loss
	TotalPnLPercentage float64 // Overall PnL percentage
	LastUpdated        string  // Report generation date
}

// BinancePriceResponse represents the response from Binance API
type BinancePriceResponse struct {
	Symbol string `json:"symbol"`
	Price  string `json:"price"`
}

type PortfolioAsset struct {
	Pair        string
	TotalAmount float64
	TotalUSD    float64
}

type PortfolioSummary struct {
	Name   string
	Assets []PortfolioAsset
}

func (e *PriceDataError) Error() string {
	return e.Message
}

// Additional types for price data fetching
type PriceDataError struct {
	InvalidPairs []string // Pairs that couldn't be fetched
	Message      string   // Error message
}

type BinanceErrorResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}
