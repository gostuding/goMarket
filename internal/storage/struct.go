package storage

import (
	"fmt"

	"gorm.io/gorm"
)

type Users struct {
	gorm.Model
	Login     string `gorm:"unique,type:varchar(20)"`
	Pwd       string `gorm:"type:varchar(32)"`
	UserAgent string `gorm:"type:varchar(255)"`
	IP        string `gorm:"type:varchar(15)"`
}

func structCheck(con *gorm.DB) error {
	err := con.AutoMigrate(&Users{})
	if err != nil {
		return fmt.Errorf("database structure error: %w", err)
	}
	return nil
}
