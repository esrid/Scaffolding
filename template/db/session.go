package db

import (
	"context"
	"fmt"
	"net"
	"time"
)

const sessionAttributes = "user_id, token,created_at,expires_at, ip_address, user_agent "

type Session struct {
	UserID    string
	Token     string
	CreatedAt time.Time
	ExpiresAt *time.Time
	IPAddress net.IP
	UserAgent string
}

type SessionStore interface {
	CreateSession(ctx context.Context, s Session) (string, error)
	GetByCookieHash(ctx context.Context, cookieHash string) (Session, error)
	DeleteByCookieHash(ctx context.Context, cookieHash string) error
	UpdateExpiry(ctx context.Context, cookieHash string, expiresAt time.Time) error
	DeleteByUserID(ctx context.Context, userID string) error
}

func (ss *PostgresStore) CreateSession(ctx context.Context, s Session) (string, error) {
	var cookieHash string
	query := fmt.Sprintf(`INSERT INTO sessions (%s) VALUES ($1, $2, NOW(), $3, $4,$5) RETURNING token`, sessionAttributes)
	if err := ss.DB.QueryRowContext(ctx, query, s.UserID, s.Token, s.ExpiresAt.UTC(), s.IPAddress.String(), s.UserAgent).Scan(&cookieHash); err != nil {
		return "", err
	}
	return cookieHash, nil
}

func (ss *PostgresStore) GetByCookieHash(ctx context.Context, cookieHash string) (Session, error) {
	var s Session
	query := fmt.Sprintf(`SELECT %s FROM sessions WHERE token = $1`, sessionAttributes)
	if err := ss.DB.QueryRowContext(ctx, query, cookieHash).Scan(&s.UserID, &s.Token, &s.CreatedAt, &s.ExpiresAt, &s.IPAddress, &s.UserAgent); err != nil {
		return Session{}, err
	}
	return s, nil
}

func (ss *PostgresStore) DeleteByCookieHash(ctx context.Context, cookieHash string) error {
	_, err := ss.DB.ExecContext(ctx, `DELETE FROM sessions WHERE token = $1`, cookieHash)
	return err
}

func (ss *PostgresStore) UpdateExpiry(ctx context.Context, cookieHash string, expiresAt time.Time) error {
	_, err := ss.DB.ExecContext(ctx, `UPDATE sessions SET expires_at = $1 WHERE token = $2 `, expiresAt, cookieHash)
	return err
}

func (ss *PostgresStore) DeleteByUserID(ctx context.Context, userID string) error {
	_, err := ss.DB.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = $1`, userID)
	return err
}
