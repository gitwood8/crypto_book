package store

import (
	"context"
	"fmt"
	"time"

	t "gitlab.com/avolkov/wood_post/pkg/types"
)

func (s *Store) AddNewTransaction(ctx context.Context, dbUserID int64, defID int, tx *t.TempTransactionData) error {

	query, args, err := s.sqlBuilder.
		Insert("transactions").
		Columns("portfolio_id", "pair", "asset_amount", "asset_price", "amount_usd", "transaction_date", "created_at").
		Values(
			defID,
			tx.Pair,
			tx.AssetAmount,
			tx.AssetPrice,
			tx.USDAmount,
			tx.TransactionDate,
			time.Now(),
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("build add new transaction query: %w", err)
	}

	_, err = s.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("exec add new transaction query: %w", err)
	}

	return nil // will be fixed
}
