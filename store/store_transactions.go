package store

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	t "gitlab.com/avolkov/wood_post/pkg/types"
)

func (s *Store) AddNewTransaction(
	ctx context.Context,
	dbUserID int64,
	defID int,
	tx *t.TempTransactionData,
) error {
	query, args, err := s.sqlBuilder.
		Insert("transactions").
		Columns(
			"portfolio_id",
			"pair",
			"asset_amount",
			"asset_price",
			"amount_usd",
			"transaction_date",
			"type",
			"created_at",
			// "note",
		).
		Values(
			defID,
			tx.Pair,
			tx.AssetAmount,
			tx.AssetPrice,
			tx.USDAmount,
			tx.TransactionDate,
			tx.Type,
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
		Where(sq.Eq{
			"p.user_id": dbUserID,
		}).
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

// FIXME
// GetLast5TransactionsForUser retrieves the last 5 transactions for a user with portfolio information
func (s *Store) GetLast5TransactionsForUser(ctx context.Context, dbUserID int64) ([]t.Transaction, error) {
	query, args, err := s.sqlBuilder.
		Select(
			"t.id",
			"p.name as portfolio_name",
			"t.type",
			"t.pair",
			"t.asset_amount",
			"t.asset_price",
			"t.amount_usd",
			"t.transaction_date",
			// "COALESCE(t.note, '') as note",
			// "t.created_at",
		).
		From("transactions t").
		LeftJoin("portfolios p ON p.id = t.portfolio_id").
		Where(sq.Eq{
			"p.user_id": dbUserID,
		}).
		OrderBy("t.created_at DESC").
		Limit(5).
		ToSql()

	if err != nil {
		return nil, fmt.Errorf("build get last 5 transactions query: %w", err)
	}

	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("exec get last 5 transactions query: %w", err)
	}
	defer rows.Close()

	var transactions []t.Transaction
	for rows.Next() {
		var tx t.Transaction
		if err := rows.Scan(
			&tx.ID,
			&tx.PortfolioName,
			&tx.Type,
			&tx.Pair,
			&tx.AssetAmount,
			&tx.AssetPrice,
			&tx.USDAmount,
			&tx.TransactionDate,
			// &tx.Note,
			// &tx.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan transaction: %w", err)
		}
		transactions = append(transactions, tx)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return transactions, nil
}

// GetPortfolioSummariesForUser retrieves portfolio summaries with asset totals for a user
func (s *Store) GetPortfolioSummariesForUser(ctx context.Context, dbUserID int64) ([]t.PortfolioSummary, error) {
	query, args, err := s.sqlBuilder.
		Select(
			"p.name as portfolio_name",
			"t.pair",
			"SUM(CASE WHEN t.type = 'buy' THEN t.asset_amount ELSE -t.asset_amount END) as total_amount",
			"SUM(CASE WHEN t.type = 'buy' THEN t.amount_usd ELSE -t.amount_usd END) as total_usd",
		).
		From("transactions t").
		InnerJoin("portfolios p ON p.id = t.portfolio_id").
		Where(sq.Eq{
			"p.user_id": dbUserID,
		}).
		GroupBy("p.name", "t.pair").
		Having("SUM(CASE WHEN t.type = 'buy' THEN t.asset_amount ELSE -t.asset_amount END) > 0").
		OrderBy("p.name", "t.pair").
		ToSql()

	if err != nil {
		return nil, fmt.Errorf("build portfolio summaries query: %w", err)
	}

	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("exec portfolio summaries query: %w", err)
	}
	defer rows.Close()

	// Use a map to group assets by portfolio
	portfolioMap := make(map[string][]t.PortfolioAsset)

	for rows.Next() {
		var portfolioName, pair string
		var totalAmount, totalUSD float64

		if err := rows.Scan(&portfolioName, &pair, &totalAmount, &totalUSD); err != nil {
			return nil, fmt.Errorf("scan portfolio summary: %w", err)
		}

		asset := t.PortfolioAsset{
			Pair:        pair,
			TotalAmount: totalAmount,
			TotalUSD:    totalUSD,
		}

		portfolioMap[portfolioName] = append(portfolioMap[portfolioName], asset)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	// Convert map to slice
	var summaries []t.PortfolioSummary
	for name, assets := range portfolioMap {
		summaries = append(summaries, t.PortfolioSummary{
			Name:   name,
			Assets: assets,
		})
	}

	return summaries, nil
}

// GetReportData retrieves aggregated transaction data across all portfolios for PnL calculations
func (s *Store) GetReportData(ctx context.Context, dbUserID int64) ([]t.CurrencyPnLData, error) {
	query, args, err := s.sqlBuilder.
		Select(
			"t.pair",
			"SUM(CASE WHEN t.type = 'buy' THEN t.asset_amount ELSE -t.asset_amount END) as total_asset_amount",
			"SUM(CASE WHEN t.type = 'buy' THEN t.amount_usd ELSE 0 END) as total_invested_usd",
		).
		From("transactions t").
		InnerJoin("portfolios p ON p.id = t.portfolio_id").
		Where(sq.Eq{
			"p.user_id": dbUserID,
		}).
		GroupBy("t.pair").
		Having("SUM(CASE WHEN t.type = 'buy' THEN t.asset_amount ELSE -t.asset_amount END) > 0").
		OrderBy("t.pair").
		ToSql()

	if err != nil {
		return nil, fmt.Errorf("build report data query: %w", err)
	}

	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("exec report data query: %w", err)
	}
	defer rows.Close()

	var reportData []t.CurrencyPnLData
	for rows.Next() {
		var data t.CurrencyPnLData
		if err := rows.Scan(
			&data.Pair,
			&data.TotalAssetAmount,
			&data.TotalInvestedUSD,
		); err != nil {
			return nil, fmt.Errorf("scan report data: %w", err)
		}

		// Calculate average purchase price
		if data.TotalAssetAmount > 0 {
			data.AveragePurchasePrice = data.TotalInvestedUSD / data.TotalAssetAmount
		}

		reportData = append(reportData, data)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return reportData, nil
}
