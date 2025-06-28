package types

import "time"

type TempTransactionData struct {
	Pair            string
	Type            string
	AssetAmount     float64
	AssetPrice      float64
	USDAmount       float64
	TransactionDate time.Time
}

// Transaction represents a complete transaction for display purposes
type Transaction struct {
	ID              int64
	PortfolioName   string
	Type            string
	Pair            string
	AssetAmount     float64
	AssetPrice      float64
	USDAmount       float64
	TransactionDate time.Time
	Note            string
	CreatedAt       time.Time
}

var DefaultCryptoPairs = []string{
	"BTCUSDT",
	"ETHUSDT",
	"DOGEUSDT",
}
