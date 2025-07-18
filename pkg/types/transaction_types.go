package types

import "time"

type TempTransactionData struct {
	ID              int64
	Asset           string
	Type            string
	AssetAmount     float64
	AssetPrice      float64
	USDAmount       float64
	TransactionDate time.Time
}

// represents a complete transaction for display purposes
type Transaction struct {
	ID              int64
	PortfolioName   string
	Type            string
	Asset           string
	AssetAmount     float64
	AssetPrice      float64
	USDAmount       float64
	TransactionDate time.Time
	Note            string
	CreatedAt       time.Time
}

var DefaultCryptoPairs = []string{
	"BTC",
	"ETH",
	"DOGE",
	"XRP",
}
