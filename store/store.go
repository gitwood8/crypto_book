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
	query, args, err := s.sqlBuilder.
		Insert("portfolios").
		Columns("user_id", "name", "description", "created_at").
		Values(userID, name, description, time.Now()).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert portfolio: %w", err)
	}

	_, err = s.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("exec insert portfolio: %w", err)
	}

	log.Infof("portfolio for name: %s, userID:%d created successfully", name, userID)

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
