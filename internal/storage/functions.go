package storage

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Users struct {
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
	Login     string    `gorm:"unique" json:"-"`
	Pwd       string    `gorm:"type:varchar(32)" json:"-"`
	UserAgent string    `gorm:"type:varchar(255)" json:"-"`
	IP        string    `gorm:"type:varchar(15)" json:"-"`
	Balance   float32   `gorm:"type:numeric" json:"curent"`
	Withdrawn float32   `gorm:"type:numeric" json:"withdrawn"`
	ID        uint      `gorm:"primarykey" json:"-"`
}

type Orders struct {
	CreatedAt time.Time `json:"uploaded_at"`
	UpdatedAt time.Time `json:"-"`
	Number    string    `gorm:"unique" json:"number"`
	Status    string    `gorm:"type:varchar(10)" json:"status"`
	Accrual   float32   `gorm:"type:numeric" json:"accrual,omitempty"`
	ID        uint      `gorm:"primarykey" json:"-"`
	UID       int       `gorm:"type:int" json:"-"`
}

type Withdraws struct {
	CreatedAt time.Time `json:"processed_at"`
	Number    string    `gorm:"unique" json:"order"`
	Sum       float32   `gorm:"type:numeric" json:"sum"`
	ID        uint      `gorm:"primarykey" json:"-"`
	UID       int       `gorm:"type:int" json:"-"`
}

func structCheck(con *gorm.DB) error {
	err := con.AutoMigrate(&Users{}, &Orders{}, &Withdraws{})
	if err != nil {
		return fmt.Errorf("database structure error: %w", err)
	}
	return nil
}
