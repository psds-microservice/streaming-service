package model

import "time"

// StreamingSession — сущность сессии трансляции (GORM).
type StreamingSession struct {
	ID         string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ClientID   string     `gorm:"type:uuid;not null;index"`
	StreamKey  string     `gorm:"size:64;not null;uniqueIndex"`
	Status     string     `gorm:"size:20;not null;default:waiting"` // waiting, active, finished
	CreatedAt  time.Time  `gorm:"autoCreateTime"`
	UpdatedAt  time.Time  `gorm:"autoUpdateTime"`
	FinishedAt *time.Time `gorm:"column:finished_at"`

	Operators []SessionOperator `gorm:"foreignKey:SessionID"`
}

func (StreamingSession) TableName() string { return "streaming_sessions" }

// SessionOperator — оператор, подключённый к сессии (GORM).
type SessionOperator struct {
	ID          string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	SessionID   string    `gorm:"type:uuid;not null;index"`
	UserID      string    `gorm:"type:uuid;not null;index"`
	ConnectedAt time.Time `gorm:"column:connected_at;not null"`
}

func (SessionOperator) TableName() string { return "session_operators" }
