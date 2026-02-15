package service

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/psds-microservice/streaming-service/internal/config"
	"github.com/psds-microservice/streaming-service/internal/errs"
	"github.com/psds-microservice/streaming-service/internal/model"
	"gorm.io/gorm"
)

// SessionService manages streaming session lifecycle.
type SessionService struct {
	db     *gorm.DB
	cfg    *config.Config
	stream *StreamHub
}

// NewSessionService creates a session service.
func NewSessionService(db *gorm.DB, cfg *config.Config, hub *StreamHub) *SessionService {
	return &SessionService{db: db, cfg: cfg, stream: hub}
}

// Create creates a new streaming session for the client.
func (s *SessionService) Create(clientID string) (*model.Session, error) {
	ent := &model.StreamingSession{
		ID:        uuid.New().String(),
		ClientID:  clientID,
		StreamKey: "sk_" + uuid.New().String()[:16],
		Status:    string(model.SessionStatusWaiting),
	}
	if err := s.db.Create(ent).Error; err != nil {
		return nil, err
	}
	return entityToSession(ent), nil
}

// Get returns a session by ID.
func (s *SessionService) Get(sessionID string) (*model.Session, error) {
	var ent model.StreamingSession
	if err := s.db.Preload("Operators").Where("id = ?", sessionID).First(&ent).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrSessionNotFound
		}
		return nil, err
	}
	return entityToSession(&ent), nil
}

// Finish marks session as finished and notifies hub.
func (s *SessionService) Finish(sessionID string) error {
	var ent model.StreamingSession
	if err := s.db.Where("id = ?", sessionID).First(&ent).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.ErrSessionNotFound
		}
		return err
	}
	now := time.Now()
	if err := s.db.Model(&ent).Updates(map[string]interface{}{
		"status":      string(model.SessionStatusFinished),
		"finished_at": now,
	}).Error; err != nil {
		return err
	}
	s.stream.CloseSession(sessionID)
	return nil
}

// AddOperator adds an operator to the session (called when operator joins WS).
func (s *SessionService) AddOperator(sessionID, userID string) error {
	var ent model.StreamingSession
	if err := s.db.Preload("Operators").Where("id = ?", sessionID).First(&ent).Error; err != nil {
		return err
	}
	if ent.Status == string(model.SessionStatusFinished) {
		return errs.ErrSessionNotFound
	}
	for _, op := range ent.Operators {
		if op.UserID == userID {
			return nil
		}
	}
	if len(ent.Operators) >= s.cfg.SessionMaxOperators {
		return errs.ErrTooManyOperators
	}
	op := &model.SessionOperator{
		ID:          uuid.New().String(),
		SessionID:   sessionID,
		UserID:      userID,
		ConnectedAt: time.Now(),
	}
	if err := s.db.Create(op).Error; err != nil {
		return err
	}
	if ent.Status == string(model.SessionStatusWaiting) {
		_ = s.db.Model(&ent).Update("status", string(model.SessionStatusActive))
	}
	return nil
}

// GetOperators returns operators for a session.
func (s *SessionService) GetOperators(sessionID string) ([]model.Operator, error) {
	var ent model.StreamingSession
	if err := s.db.Preload("Operators").Where("id = ?", sessionID).First(&ent).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrSessionNotFound
		}
		return nil, err
	}
	out := make([]model.Operator, 0, len(ent.Operators))
	for _, o := range ent.Operators {
		out = append(out, model.Operator{UserID: o.UserID, ConnectedAt: o.ConnectedAt})
	}
	return out, nil
}

func entityToSession(ent *model.StreamingSession) *model.Session {
	sess := &model.Session{
		ID:         ent.ID,
		ClientID:   ent.ClientID,
		StreamKey:  ent.StreamKey,
		Status:     model.SessionStatus(ent.Status),
		CreatedAt:  ent.CreatedAt,
		FinishedAt: ent.FinishedAt,
	}
	for _, o := range ent.Operators {
		sess.Operators = append(sess.Operators, model.Operator{UserID: o.UserID, ConnectedAt: o.ConnectedAt})
	}
	return sess
}
