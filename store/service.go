package store

import (
	"database/sql"
	"fmt"
	"time"

	"gitlab.com/avolkov/wood_post/pkg/log"

	sq "github.com/Masterminds/squirrel"
	_ "github.com/lib/pq"
)

type Store struct {
	DB         *sql.DB
	sqlBuilder sq.StatementBuilderType // SQL query builder from squirrel
	TempName   map[int64]string        // temp storage for portfolio name per user
	// moved to session manager
	// mu         sync.RWMutex            // cuncurency protection while multiple users interaction
}

// establish DB connection
func New(user, password, host, port, dbname string) (*Store, error) {
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		user, password, host, port, dbname,
	)
	log.Info(connStr)

	// log.Info(connStr)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("db open connection error: %w", err)
	}

	// if err := db.Ping(); err != nil {
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
