package storage

import (
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type psqlStorage struct {
	con    *gorm.DB
	logger *zap.SugaredLogger
}

func NewPSQLStorage(connectionString string, logger *zap.SugaredLogger) (*psqlStorage, error) {
	db, err := gorm.Open(postgres.Open(connectionString), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("database connection create error: %w", err)
	}
	storage := psqlStorage{
		con:    db,
		logger: logger,
	}
	return &storage, structCheck(db)
}
