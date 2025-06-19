package store

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
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

func (s *Store) GetTopPairsForUser(ctx context.Context, dbUserID int64) ([]string, error) {
	query, args, err := s.sqlBuilder.
		Select("t.pair").
		From("transactions t").
		LeftJoin("portfolios p ON p.id = t.portfolio_id").
		Where(sq.Eq{"p.user_id": dbUserID}).
		GroupBy("p.user_id, t.pair").
		OrderBy("COUNT(t.pair) DESC").
		Limit(5).
		ToSql()

	if err != nil {
		return nil, fmt.Errorf("build top pairs query: %w", err)
	}

	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("exec top pairs query: %w", err)
	}
	defer rows.Close()

	var pairs []string
	for rows.Next() {
		var pair string
		if err := rows.Scan(&pair); err != nil {
			return nil, fmt.Errorf("scan top pair: %w", err)
		}
		pairs = append(pairs, pair)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return pairs, nil
}
