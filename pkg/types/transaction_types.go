package types

import "time"

type TempTransactionData struct {
	Pair            string
	AssetAmount     float64
	AssetPrice      float64
	USDAmount       float64
	TransactionDate time.Time
	// Note            string
}
