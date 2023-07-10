package storage

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Users struct {
	ID        uint      `gorm:"primarykey" json:"-"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
	Login     string    `gorm:"unique,type:varchar(20)" json:"-"`
	Pwd       string    `gorm:"type:varchar(32)" json:"-"`
	UserAgent string    `gorm:"type:varchar(255)" json:"-"`
	IP        string    `gorm:"type:varchar(15)" json:"-"`
	Balance   float32   `gorm:"type:double precision" json:"curent"`
	Withdrawn float32   `gorm:"type:double precision" json:"withdrawn"`
}

type Orders struct {
	ID        uint      `gorm:"primarykey" json:"-"`
	CreatedAt time.Time `json:"uploaded_at"`
	UpdatedAt time.Time `json:"-"`
	Number    string    `gorm:"unique" json:"number"`
	UID       int       `gorm:"type:int" json:"-"`
	Status    string    `gorm:"type:varchar(10)" json:"status"`
	Accrual   float32   `gorm:"type:double precision" json:"accrual,omitempty"`
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
