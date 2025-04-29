package store

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"gitlab.com/avolkov/wood_post/pkg/log"

	sq "github.com/Masterminds/squirrel"
	_ "github.com/lib/pq"
)

type Store struct {
	DB         *sql.DB
	sqlBuilder sq.StatementBuilderType // SQL query builder from squirrel
	TempName   map[int64]string        // temp storage for portfolio name per user
	mu         sync.RWMutex            // cuncurency protection while multiple users interaction
}

// establish DB connection
func New(user, password, host, port, dbname string) (*Store, error) {
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		user, password, host, port, dbname,
	)

	// log.Info(connStr)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("db open connection error: %w", err)
	}

	// if err := db.Ping(); err != nil {:::::::
	// 	return nil, fmt.Errorf("db ping error: %w", err)
	// }

	for i := 0; i < 5; i++ {
		err = db.Ping()
		if err == nil {
			// log.Info("db is ready")
			break
		}
		log.Warn("Database not ready yet, retrying in 2s...")
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		return nil, fmt.Errorf("cannot connect to database after retries: %w", err)
	}

	log.Info("store: connected to database")

	return &Store{
		DB:         db,
		sqlBuilder: sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
		TempName:   make(map[int64]string),
	}, nil
}

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

	log.Infof("user name: %s, tgID:%d created successfully", username, telegramID)

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

// Безопасная работа с TempName

// func (s *Store) SetTempName(userID int64, name string) {
// 	s.mu.Lock()
// 	defer s.mu.Unlock()
// 	s.TempName[userID] = name
// }

// func (s *Store) GetTempName(userID int64) (string, bool) {
// 	s.mu.RLock()
// 	defer s.mu.RUnlock()
// 	name, ok := s.TempName[userID]
// 	return name, ok
// }

// func (s *Store) ClearTempName(userID int64) {
// 	s.mu.Lock()
// 	defer s.mu.Unlock()
// 	delete(s.TempName, userID)
// }
