package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"
	"{{projectName}}/db"
	"{{projectName}}/utils"

	"github.com/resend/resend-go/v2"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrSessionExpired      = errors.New("session has expired")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidSession      = errors.New("invalid session")
	ErrInvalidEmailFormat  = errors.New("invalid email format")
	ErrPasswordTooWeak     = errors.New("password must be at least 8 characters")
	ErrEmailAlreadyInUse   = errors.New("email already in use")
	ErrPasswordHashFailed  = errors.New("failed to hash password")
	ErrOTPGenerationFailed = errors.New("failed to generate OTP code")
	ErrInvalidOTPCode      = errors.New("invalid OTP code")
	ErrOTPExpired          = errors.New("OTP code has expired")
	ErrEmailSendFailed     = errors.New("failed to send email")
)

const sessionDuration = 24 * time.Hour // Default session duration

func GetIPAddressBytes(r *http.Request) []byte {
	xForwardedFor := r.Header.Get("X-Forwarded-For")
	if xForwardedFor != "" {
		ips := strings.Split(xForwardedFor, ",")
		ip := strings.TrimSpace(ips[0])
		parsedIP := net.ParseIP(ip)
		if parsedIP != nil {
			return parsedIP
		}
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		parsedIP := net.ParseIP(ip)
		if parsedIP != nil {
			return parsedIP
		}
	}
	return nil
}

func GenerateSessionToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func CheckPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// ValidateEmail checks if the email format is valid
func ValidateEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

func IsValidPasswordLength(password string) bool {
	length := len([]byte(password)) // counts bytes, not runes
	return length >= 8 && length <= 72
}

func ValidateUserInput(user db.User) error {
	if !ValidateEmail(user.Email) {
		return ErrInvalidEmailFormat
	}
	if !IsValidPasswordLength(user.PasswordHash) {
		return ErrPasswordTooWeak
	}
	return nil
}

func RegisterUser(ctx context.Context, store db.AuthStore, user db.User) (*db.User, error) {
	if err := ValidateUserInput(user); err != nil {
		return nil, err
	}

	_, err := store.GetUserByEmail(ctx, user.Email)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	user.PasswordHash, err = utils.HashPassword(user.PasswordHash)
	if err != nil {
		return nil, ErrPasswordHashFailed
	}

	return store.CreateUser(ctx, &user)
}

func LoginUser(ctx context.Context, s db.Store, u db.User) (*db.User, error) {
	if err := ValidateUserInput(u); err != nil {
		return nil, err
	}

	existing, err := s.GetUserByEmail(ctx, u.Email)
	if err != nil {
		return nil, err
	}

	if err := CheckPassword(existing.PasswordHash, u.PasswordHash); err != nil {
		return nil, ErrInvalidCredentials
	}

	return existing, nil
}

func CreateSession(ctx context.Context, ss db.SessionStore, userID string, r *http.Request) (string, error) {
	// Generate secure session token
	cookieHash, err := GenerateSessionToken()
	if err != nil {
		return "", err
	}

	session := db.Session{
		UserID:    userID,
		Token:     cookieHash,
		IPAddress: GetIPAddressBytes(r),
		UserAgent: r.UserAgent(),
	}

	// Set expiration time
	expiresAt := time.Now().Add(sessionDuration)
	session.ExpiresAt = &expiresAt

	// Create the session

	return ss.CreateSession(ctx, session)
}

// ValidateSession validates a session and checks for expiration
func ValidateSession(ctx context.Context, cookieHash string, ss db.SessionStore) (db.Session, error) {
	session, err := ss.GetByCookieHash(ctx, cookieHash)
	if err != nil {
		return db.Session{}, ErrInvalidSession
	}

	// Check if session has expired
	if session.ExpiresAt != nil && time.Now().After(*session.ExpiresAt) {
		// Automatically delete expired session
		_ = ss.DeleteByCookieHash(ctx, cookieHash)
		return db.Session{}, ErrSessionExpired
	}

	return session, nil
}

// RefreshSession extends the session duration
func RefreshSession(ctx context.Context, cookieHash string, ss db.SessionStore) error {
	// Validate session first
	_, err := ValidateSession(ctx, cookieHash, ss)
	if err != nil {
		return err
	}

	// Set new expiration time
	newExpiryTime := time.Now().Add(sessionDuration)
	return ss.UpdateExpiry(ctx, cookieHash, newExpiryTime)
}

// RevokeSession invalidates a session
func RevokeSession(ctx context.Context, cookieHash string, ss db.SessionStore) error {
	return ss.DeleteByCookieHash(ctx, cookieHash)
}

// RevokeAllUserSessions invalidates all sessions for a given user
func RevokeAllUserSessions(ctx context.Context, userID string, ss db.SessionStore) error {
	return ss.DeleteByUserID(ctx, userID)
}

func generateSecureOTP(length int) (int, error) {
	if length <= 0 || length > 9 {
		return 0, errors.New("length should be between 0 to 9 include")
	}

	var otpChars strings.Builder
	otpChars.Grow(length)

	firstDigitLimit := big.NewInt(9)
	firstDigit, err := rand.Int(rand.Reader, firstDigitLimit)
	if err != nil {
		return 0, err
	}
	firstDigit.Add(firstDigit, big.NewInt(1))
	otpChars.WriteString(firstDigit.String())

	digitLimit := big.NewInt(10)
	for i := 1; i < length; i++ {
		digit, err2 := rand.Int(rand.Reader, digitLimit)
		if err2 != nil {
			return 0, err
		}
		otpChars.WriteString(digit.String())
	}

	otpStr := otpChars.String()
	toint, err := strconv.Atoi(otpStr)
	if err != nil {
		return 0, fmt.Errorf("échec de la conversion en uint32: %w", err)
	}

	return toint, nil
}

func CreateOTP(ctx context.Context, store db.OtpStore, id string, api string) error {
	code, err := generateSecureOTP(6)
	if err != nil {
		return ErrOTPGenerationFailed
	}
	if err := store.CreateOtp(ctx, &db.Otp{Code: code, UserId: id, CreatedAt: time.Now(), Used: false}); err != nil {
		return err
	}

	client := resend.NewClient(api)
	params := &resend.SendEmailRequest{
		From:    "",
		To:      []string{""},
		Subject: "",
		Text:    fmt.Sprintf("your code is", code),
	}

	if _, err := client.Emails.Send(params); err != nil {
		return ErrEmailSendFailed
	}

	return nil
}

func ValidateOTP(ctx context.Context, userId string, code string, store db.OtpStore) error {
	intcode, err := strconv.Atoi(code)
	if err != nil {
		return err
	}
	otp, err := store.GetOtp(ctx, userId, intcode)
	if err != nil {
		return ErrInvalidOTPCode
	}

	if time.Since(otp.CreatedAt) > time.Duration(5*time.Minute) {
		return ErrOTPExpired
	}

	if otp.Used {
		return ErrInvalidOTPCode
	}

	if err := store.MarkOtpAsUsed(ctx, userId, intcode); err != nil {
		return fmt.Errorf("failed to mark OTP as used: %w", err)
	}

	return nil
}
