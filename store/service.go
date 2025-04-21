package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	_ "github.com/lib/pq"
)

type Store struct {
	DB *sql.DB
	q  sq.StatementBuilderType
}

func New(user, password, host, port, dbname string) (*Store, error) {
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		user, password, host, port, dbname,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ошибка проверки соединения: %w", err)
	}

	return &Store{
		DB: db,
		q:  sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}, nil
}

// Purchase — модель покупки
type Purchase struct {
	UserID    int64
	Pair      string
	BoughtAt  time.Time
	AmountUSD float64
}

// AddPurchase сохраняет новую покупку в БД
func (s *Store) AddPurchase(ctx context.Context, p *Purchase) error {
	query, args, err := s.q.
		Insert("purchases").
		Columns("user_id", "pair", "bought_at", "amount_usd").
		Values(p.UserID, p.Pair, p.BoughtAt, p.AmountUSD).
		ToSql()
	if err != nil {
		return fmt.Errorf("ошибка сборки запроса: %w", err)
	}

	_, err = s.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("ошибка выполнения запроса: %w", err)
	}

	return nil
}
