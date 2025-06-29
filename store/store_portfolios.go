package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"gitlab.com/avolkov/wood_post/pkg/log"
)

func (s *Store) CreatePortfolio(ctx context.Context, dbUserID int64, portfolioName string, description string) error {
	exists, err := s.PortfolioExists(ctx, dbUserID)
	if err != nil {
		return fmt.Errorf("failed to check portfolio existence: %w", err)
	}

	isDefault := !exists

	query, args, err := s.sqlBuilder.
		Insert("portfolios").
		Columns("user_id", "name", "description", "is_default", "created_at").
		Values(dbUserID, portfolioName, description, isDefault, time.Now()).
		ToSql()
	if err != nil {
		return fmt.Errorf("build PortfolioExists query: %w", err)
	}

	_, err = s.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("exec PortfolioExists query: %w", err)
	}

	log.Infof("portfolio for userID:%d with name: %s created (is_default: %v)", dbUserID, portfolioName, isDefault)
	return nil
}

// without sqlBuilder example:
// func (s *Store) GetUserIDByTelegramID(ctx context.Context, telegramID int64) (int64, error) {
// 	var id int64
// 	err := s.DB.QueryRowContext(ctx, "SELECT id FROM users WHERE telegram_id = $1", telegramID).Scan(&id)
// 	if err != nil {
// 		return 0, fmt.Errorf("get user id by telegram_id: %w", err)
// 	}
// 	return id, nil
// }

func (s *Store) PortfolioExists(ctx context.Context, dbUserID int64) (bool, error) {
	// only check if row exists, no full scan, no data from table
	query, args, err := s.sqlBuilder.
		Select("1").
		From("portfolios").
		Where(sq.Eq{
			"user_id": dbUserID,
		}).
		Limit(1).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build PortfolioExists query: %w", err)
	}

	var exists int
	err = s.DB.QueryRowContext(ctx, query, args...).Scan(&exists)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("exec PortfolioExists query: %w", err)
	}

	return true, nil
}

func (s *Store) ReachedPortfolioLimit(ctx context.Context, dbUserID int64) (bool, error) {
	var count int
	query, args, err := s.sqlBuilder.
		Select("COUNT(*)").
		From("portfolios").
		Where(sq.Eq{
			"user_id": dbUserID,
		}).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build ReachedPortfolioLimit query: %w", err)
	}
	if err := s.DB.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return false, fmt.Errorf("exec ReachedPortfolioLimit query: %w", err)
	}
	return count >= 2, nil
}

func (s *Store) PortfolioNameExists(ctx context.Context, dbUserID int64, portfolioName string) (bool, error) {
	query, args, err := s.sqlBuilder.
		Select("1").
		From("portfolios").
		Where(sq.Eq{
			"user_id": dbUserID,
			"name":    portfolioName,
		}).
		Limit(1).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build PortfolioNameExists query: %w", err)
	}

	var exists int
	err = s.DB.QueryRowContext(ctx, query, args...).Scan(&exists)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	// any other error
	return false, fmt.Errorf("exec PortfolioNameExists query: %w", err)
}

// func (db *sql.DB) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row

func (s *Store) DeletePortfolio(ctx context.Context, dbUserID int64, portfolioName string) error {
	query, args, err := s.sqlBuilder.
		Delete("portfolios").
		Where(sq.Eq{
			"user_id": dbUserID,
			"name":    portfolioName,
		}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete query: %w", err)
	}

	result, err := s.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("exec delete query: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		log.Warnf("no portfolio deleted for user_id=%d, portfolio_name=%s", dbUserID, portfolioName)
	}

	return nil
}

func (s *Store) GetDefaultPortfolio(ctx context.Context, dbUserID int64) (string, error) {
	query, args, err := s.sqlBuilder.
		Select("name").
		From("portfolios").
		Where(sq.Eq{
			"user_id":    dbUserID,
			"is_default": true,
		}).
		ToSql()
	if err != nil {
		return "", fmt.Errorf("build GetDefaultPortfolio query: %w", err)
	}

	fmt.Println("ASDASDASDASDASD")

	var dpName string
	err = s.DB.QueryRowContext(ctx, query, args...).Scan(&dpName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "ХУЙ", err
		}
		return "", fmt.Errorf("exec GetDefaultPortfolio query: %w", err)
	}

	fmt.Println("pLplPlpplplplp")
	return dpName, nil
}

func (s *Store) GetDefaultPortfolioID(ctx context.Context, dbUserID int64) (int, error) {
	query, args, err := s.sqlBuilder.
		Select("id").
		From("portfolios").
		Where(sq.Eq{
			"user_id":    dbUserID,
			"is_default": true,
		}).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("build GetDefaultPortfolioID query: %w", err)
	}

	var dpID int
	err = s.DB.QueryRowContext(ctx, query, args...).Scan(&dpID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, err
		}
		return 0, fmt.Errorf("exec GetDefaultPortfolioID query: %w", err)
	}
	return dpID, nil
}

func (s *Store) GetPortfoliosFiltered(
	ctx context.Context,
	dbUserID int64,
	onlyNonDefault bool,
) ([]string, error) {
	builder := s.sqlBuilder.
		Select("name").
		From("portfolios").
		Where(sq.Eq{"user_id": dbUserID})

	if onlyNonDefault {
		builder = builder.Where(sq.Eq{"is_default": false})
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build GetPortfoliosFiltered query: %w", err)
	}

	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("exec GetPortfoliosFiltered query: %w", err)
	}
	defer rows.Close()

	var portfolios []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan GetPortfoliosFiltered result: %w", err)
		}
		portfolios = append(portfolios, name)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return portfolios, nil
}

// TODO: add check for name existance
func (s *Store) RenamePortfolio(
	ctx context.Context,
	dbUserID int64,
	oldName, newName string,
) error {
	query, args, err := s.sqlBuilder.
		Update("portfolios").
		Set("name", newName).
		Where(sq.Eq{
			"user_id": dbUserID,
			"name":    oldName,
		}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build RenamePortfolio query: %w", err)
	}

	result, err := s.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("exec RenamePortfolio query: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("rename failed: no portfolio found with name '%s'", oldName)
	}

	log.Infof("portfolio renamed: user_id=%d, from=%s to=%s", dbUserID, oldName, newName)
	return nil
}

func (s *Store) ChangeDefaultPortfolio(
	ctx context.Context,
	dbUserID int64,
	portfolioName string,
) error {
	resetQuery, resetArgs, err := s.sqlBuilder.
		Update("portfolios").
		Set("is_default", false).
		Where(sq.Eq{
			"user_id": dbUserID,
		}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build reset default query: %w", err)
	}

	_, err = s.DB.ExecContext(ctx, resetQuery, resetArgs...)
	if err != nil {
		return fmt.Errorf("exec reset default query: %w", err)
	}

	setQuery, setArgs, err := s.sqlBuilder.
		Update("portfolios").
		Set("is_default", true).
		Where(sq.Eq{
			"user_id": dbUserID,
			"name":    portfolioName,
		}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build set default query: %w", err)
	}

	result, err := s.DB.ExecContext(ctx, setQuery, setArgs...)
	if err != nil {
		return fmt.Errorf("exec set default query: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("set default failed: no portfolio found with name '%s'", portfolioName)
	}

	log.Infof("default portfolio changed: user_id=%d, new_default='%s'", dbUserID, portfolioName)

	return nil
}
