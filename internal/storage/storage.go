package storage

import (
	"context"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"gorm.io/gorm/logger"
)

type psqlStorage struct {
	con *gorm.DB
}

func NewPSQLStorage(connectionString string) (*psqlStorage, error) {
	db, err := gorm.Open(postgres.Open(connectionString), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		return nil, fmt.Errorf("database connection create error: %w", err)
	}
	storage := psqlStorage{
		con: db,
	}
	return &storage, structCheck(db)
}

func (s *psqlStorage) Registration(ctx context.Context, login, pwd, ua, ip string) (int, error) {
	user := Users{Login: login, Pwd: getMD5Hash(pwd), UserAgent: ua, IP: ip}
	result := s.con.WithContext(ctx).Create(&user)
	if result.Error != nil {
		return 0, fmt.Errorf("sql error: %w", result.Error)
	}
	return int(user.ID), nil
}
