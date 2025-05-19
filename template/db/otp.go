package db

import (
	"context"
	"fmt"
	"time"
)

const (
	otpAttributes = "user_id, code, created_at, used"
	expiryTime    = 5 * time.Minute
)

type Otp struct {
	Code      int
	UserId    string
	CreatedAt time.Time
	Used      bool
}

type OtpStore interface {
	CreateOtp(ctx context.Context, otp *Otp) error
	GetOtp(ctx context.Context, userId string, code int) (*Otp, error)
	MarkOtpAsUsed(ctx context.Context, userId string, code int) error
	DeleteExpiredOtps(ctx context.Context) error
}

func (r *PostgresStore) CreateOtp(ctx context.Context, otp *Otp) error {
	query := fmt.Sprintf(`INSERT INTO otps (%s) VALUES ($1, $2, $3, $4)`, otpAttributes)
	_, err := r.DB.ExecContext(ctx, query, otp.UserId, otp.Code, otp.CreatedAt, false)
	return err
}

func (r *PostgresStore) GetOtp(ctx context.Context, userId string, code int) (*Otp, error) {
	otp := &Otp{}
	query := fmt.Sprintf(`SELECT %s FROM otps WHERE user_id = $1 AND code = $2`, otpAttributes)
	if err := r.DB.QueryRowContext(ctx, query, userId, code).Scan(&otp.UserId, &otp.Code, &otp.CreatedAt, &otp.Used); err != nil {
		return nil, err
	}
	return otp, nil
}

func (r *PostgresStore) MarkOtpAsUsed(ctx context.Context, userId string, code int) error {
	query := `UPDATE otps SET used = true WHERE user_id = $1 AND code = $2`
	if _, err := r.DB.ExecContext(ctx, query, userId, code); err != nil {
		return err
	}
	return nil
}

func (r *PostgresStore) DeleteExpiredOtps(ctx context.Context) error {
	query := `DELETE FROM otps WHERE created_at < $1`
	_, err := r.DB.ExecContext(ctx, query, time.Now().Add(-expiryTime))
	return err
}
