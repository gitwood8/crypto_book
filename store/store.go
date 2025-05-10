package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/pkg/errors"

	sq "github.com/Masterminds/squirrel"
	"gitlab.com/avolkov/wood_post/pkg/log"
)

func (s *Store) CreateUserIfNotExists(ctx context.Context, telegramID int64, username string) error {
	query, args, err := s.sqlBuilder.
		Insert("users").
		Columns("telegram_id", "username").
		Values(telegramID, username).
		Suffix("ON CONFLICT (telegram_id) DO NOTHING").
		ToSql()

	if err != nil {
		return fmt.Errorf("build insert user: %w", err)
	}

	_, err = s.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("exec insert user: %w", err)
	}

	log.Infof("user name: %s, tgID: %d created successfully", username, telegramID)

	return nil
}

func (s *Store) CreatePortfolio(ctx context.Context, userID int64, name string, description string) error {
	exists, err := s.PortfolioExists(ctx, userID)
	if err != nil {
		return errors.Wrap(err, "failed to check portfolio existence")
	}

	isDefault := !exists

	query, args, err := s.sqlBuilder.
		Insert("portfolios").
		Columns("user_id", "name", "description", "is_default", "created_at").
		Values(userID, name, description, isDefault, time.Now()).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert portfolio: %w", err)
	}

	_, err = s.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("exec insert portfolio: %w", err)
	}

	log.Infof("portfolio for userID:%d with name: %s created (is_default: %v)", userID, name, isDefault)
	return nil
}

func (s *Store) GetUserIDByTelegramID(ctx context.Context, telegramID int64) (int64, error) {
	var id int64
	err := s.DB.QueryRowContext(ctx, "SELECT id FROM users WHERE telegram_id = $1", telegramID).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("get user id by telegram_id: %w", err)
	}
	return id, nil
}

func (s *Store) UserExists(ctx context.Context, telegramID int64) (bool, error) {
	// only check if row exists, no full scan, no data from table
	query, args, err := s.sqlBuilder.
		Select("1").
		From("users").
		Where(sq.Eq{"telegram_id": telegramID}).
		Limit(1).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("failed to build UserExists query: %w", err)
	}

	var exists int
	err = s.DB.QueryRowContext(ctx, query, args...).Scan(&exists)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("failed to execute UserExists query: %w", err)
	}

	return true, nil
}

func (s *Store) PortfolioExists(ctx context.Context, userID int64) (bool, error) {
	// only check if row exists, no full scan, no data from table
	query, args, err := s.sqlBuilder.
		Select("1").
		From("portfolios").
		Where(sq.Eq{"user_id": userID}).
		Limit(1).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("failed to build PortfolioExists query: %w", err)
	}

	var exists int
	err = s.DB.QueryRowContext(ctx, query, args...).Scan(&exists)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("failed to execute PortfolioExists query: %w", err)
	}

	return true, nil
}
