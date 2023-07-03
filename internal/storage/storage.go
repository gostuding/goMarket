package storage

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
)

type psqlStorage struct {
	con    *sql.DB
	logger *zap.SugaredLogger
}

func NewPSQLStorage(connectionString string, logger *zap.SugaredLogger) (*psqlStorage, error) {
	db, err := sql.Open("pgx", connectionString)
	if err != nil {
		return nil, fmt.Errorf("database connection create error: %w", err)
	}
	storage := psqlStorage{
		con:    db,
		logger: logger,
	}
	return &storage, structCheck(db)
}

func (s *psqlStorage) Ping(ctx context.Context) error {
	return s.con.PingContext(ctx)
}
