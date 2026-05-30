package domain

import "time"

type APIEndpoint struct {
	ID        uint      `json:"id" gorm:"primarykey;type:bigint unsigned"`
	Method    string    `json:"method" gorm:"type:varchar(10);not null;uniqueIndex:uk_method_path"`
	Path      string    `json:"path" gorm:"type:varchar(255);not null;uniqueIndex:uk_method_path"`
	CreatedAt time.Time `json:"created_at" gorm:"not null"`
}

func (APIEndpoint) TableName() string {
	return "api_endpoints"
}
