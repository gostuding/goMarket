package storage

import (
	"crypto/md5"
	"encoding/hex"
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

type Orders struct {
	gorm.Model
	Order  string `gorm:"unique"`
	UID    int    `gorm:"type:int"`
	Status string `gorm:"type:varchar(10)"`
}

func structCheck(con *gorm.DB) error {
	err := con.AutoMigrate(&Users{}, &Orders{})
	if err != nil {
		return fmt.Errorf("database structure error: %w", err)
	}
	return nil
}

func getMD5Hash(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}
