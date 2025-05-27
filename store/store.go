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

	log.Infof("user: %s, tgID: %d created successfully", username, telegramID)

	return nil
}

func (s *Store) CreatePortfolio(ctx context.Context, userID int64, portfolioName string, description string) error {
	exists, err := s.PortfolioExists(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to check portfolio existence: %w", err)
	}

	isDefault := !exists

	query, args, err := s.sqlBuilder.
		Insert("portfolios").
		Columns("user_id", "name", "description", "is_default", "created_at").
		Values(userID, portfolioName, description, isDefault, time.Now()).
		ToSql()
	if err != nil {
		return fmt.Errorf("build PortfolioExists query: %w", err)
	}

	_, err = s.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("exec PortfolioExists query: %w", err)
	}

	log.Infof("portfolio for userID:%d with name: %s created (is_default: %v)", userID, portfolioName, isDefault)
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

func (s *Store) GetUserIDByTelegramID(ctx context.Context, telegramID int64) (int64, error) {
	query, args, err := s.sqlBuilder.
		Select("id").
		From("users").
		Where(sq.Eq{
			"telegram_id": telegramID,
		}).
		Limit(1).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("build GetUserIDByTelegramID query: %w", err)
	}

	var id int64
	err = s.DB.QueryRowContext(ctx, query, args...).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("exec GetUserIDByTelegramID query: %w", err)
	}

	return id, nil
}

func (s *Store) UserExists(ctx context.Context, telegramID int64) (bool, error) {
	// only check if row exists, no full scan, no data from table
	query, args, err := s.sqlBuilder.
		Select("1").
		From("users").
		Where(sq.Eq{
			"telegram_id": telegramID,
		}).
		Limit(1).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build UserExists query: %w", err)
	}

	var exists int
	err = s.DB.QueryRowContext(ctx, query, args...).Scan(&exists)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("exec UserExists query: %w", err)
	}

	return true, nil
}

func (s *Store) PortfolioExists(ctx context.Context, userID int64) (bool, error) {
	// only check if row exists, no full scan, no data from table
	query, args, err := s.sqlBuilder.
		Select("1").
		From("portfolios").
		Where(sq.Eq{
			"user_id": userID,
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

func (s *Store) ReachedPortfolioLimit(ctx context.Context, userID int64) (bool, error) {
	var count int
	query, args, err := s.sqlBuilder.
		Select("COUNT(*)").
		From("portfolios").
		Where(sq.Eq{
			"user_id": userID,
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

func (s *Store) PortfolioNameExists(ctx context.Context, userID int64, portfolioName string) (bool, error) {
	query, args, err := s.sqlBuilder.
		Select("1").
		From("portfolios").
		Where(sq.Eq{
			"user_id": userID,
			"name":    portfolioName,
		}).
		Limit(1).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build unique name check query: %w", err)
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
	return false, fmt.Errorf("exec unique portfolio name check query: %w", err)
}

func (s *Store) GetPortfolios(ctx context.Context, userID int64) ([]string, error) {
	query, args, err := s.sqlBuilder.
		Select("name").
		From("portfolios").
		Where(sq.Eq{
			"user_id": userID,
		}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build GetPortfolios query: %w", err)
	}

	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("exec GetPortfolios query: %w", err)
	}
	defer rows.Close()

	var pNames []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan GetPortfolios result: %w", err)
		}
		pNames = append(pNames, name)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return pNames, nil
}

// func (db *sql.DB) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row

func (s *Store) GfDeletePortfolio(ctx context.Context, dbUserID int64, portfolioName string) error {
	// r, _ := s.sessions.getSessionVars(tgUserID)
	// fmt.Println("portfilio to delete: ", r.SelectedPortfolioName)

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
		log.Warnf("no portfolio deleted for user_id=%d, p_name=%s", dbUserID, portfolioName)
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
		if errors.Is(err, sql.ErrNoRows) {
			return "", err
		}
		return "", fmt.Errorf("build GetDefaultPortfolio query: %w", err)
	}

	var dpName string
	err = s.DB.QueryRowContext(ctx, query, args...).Scan(&dpName)
	if err != nil {
		return "", fmt.Errorf("exec GetDefaultPortfolio query: %w", err)
	}
	return dpName, nil
}

func (s *Store) RenamePortfolio(ctx context.Context, dbUserID int64, oldName, newName string) error {
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

	log.Infof("portfolio renamed: user_id=%d, from='%s' to='%s'", dbUserID, oldName, newName)
	return nil
}
