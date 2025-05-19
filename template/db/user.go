package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

const userAttribute = "id,email,password_hash,google_id,oauth,verify,role,created_at,updated_at"

type User struct {
	ID           string
	Email        string
	PasswordHash string
	GoogleID     string
	Oauth        bool
	Verify       bool
	Role         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type GoogleUser struct {
	Id            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
}

type UserStore interface {
	CreateUser(ctx context.Context, u *User) (*User, error)
	CreateUserWithGoogle(ctx context.Context, u *User) (*User, error)
	GetUserByID(ctx context.Context, id string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByGoogleID(ctx context.Context, gid string) (*User, error)
	GetAllUsers(ctx context.Context) ([]*User, error)
	UpdateUser(ctx context.Context, u *User) error
	DeleteUser(ctx context.Context, id string) error
	GetUserBySessionID(ctx context.Context, sid string) (*User, error)
	UpdateVerify(ctx context.Context, id string, verify bool) error
}

func (r *PostgresStore) CreateUser(ctx context.Context, u *User) (*User, error) {
	user := &User{}
	var google_id sql.NullString
	query := fmt.Sprintf(
		`INSERT INTO users (email, password_hash, created_at, updated_at) VALUES ($1, $2, NOW(), NOW()) RETURNING %s`,
		userAttribute,
	)

	if err := r.DB.QueryRowContext(
		ctx,
		query,
		u.Email,
		u.PasswordHash,
	).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&google_id,
		&user.Oauth,
		&user.Verify,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		return nil, err
	}
	user.GoogleID = ""
	user.PasswordHash = ""
	return user, nil
}

func (r *PostgresStore) CreateUserWithGoogle(ctx context.Context, u *User) (*User, error) {
	user := &User{}
	var google_id sql.NullString
	var password sql.NullString
	query := fmt.Sprintf(
		`INSERT INTO users (email, google_id, oauth, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW()) RETURNING %s`,
		userAttribute,
	)
	if err := r.DB.QueryRowContext(
		ctx,
		query,
		u.Email,
		u.GoogleID,
		u.Oauth,
	).Scan(
		&user.ID,
		&user.Email,
		&password,
		&google_id,
		&user.Oauth,
		&user.Verify,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		return nil, err
	}
	user.GoogleID = google_id.String
	user.PasswordHash = ""
	return user, nil
}

func (r *PostgresStore) GetUserByID(ctx context.Context, id string) (*User, error) {
	user := &User{}
	var google_id sql.NullString
	query := fmt.Sprintf(`SELECT %s FROM users WHERE id = $1`, userAttribute)
	if err := r.DB.QueryRowContext(
		ctx,
		query,
		id,
	).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&google_id,
		&user.Oauth,
		&user.Verify,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		return nil, err
	}

	user.GoogleID = google_id.String
	user.PasswordHash = ""
	return user, nil
}

func (r *PostgresStore) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	user := &User{}
	var google_id sql.NullString
	query := fmt.Sprintf(`SELECT %s FROM users WHERE email = $1`, userAttribute)
	if err := r.DB.QueryRowContext(
		ctx,
		query,
		email,
	).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&google_id,
		&user.Oauth,
		&user.Verify,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		return nil, err
	}

	user.GoogleID = google_id.String
	return user, nil
}

func (r *PostgresStore) GetUserByGoogleID(ctx context.Context, gid string) (*User, error) {
	user := &User{}
	var google_id sql.NullString
	query := fmt.Sprintf(`SELECT %s FROM users WHERE google_id = $1`, userAttribute)
	if err := r.DB.QueryRowContext(
		ctx,
		query,
		gid,
	).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&google_id,
		&user.Oauth,
		&user.Verify,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		return nil, err
	}

	user.GoogleID = google_id.String
	user.PasswordHash = ""
	return user, nil
}

func (r *PostgresStore) GetAllUsers(ctx context.Context) ([]*User, error) {
	query := fmt.Sprintf(`SELECT %s FROM users`, userAttribute)
	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		var google_id sql.NullString
		u := &User{}
		if err := rows.Scan(
			&u.ID, &u.Email, &u.PasswordHash, &google_id, &u.Oauth, &u.Verify, &u.Role, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, err
		}
		u.PasswordHash = ""
		users = append(users, u)
	}
	return users, nil
}

func (r *PostgresStore) UpdateUser(ctx context.Context, u *User) error {
	_, err := r.DB.ExecContext(ctx, `
        UPDATE users SET email = $1, password_hash = $2, google_id = $3, updated_at = NOW() WHERE id = $4`,
		u.Email, u.PasswordHash, u.GoogleID, u.ID)
	return err
}

func (r *PostgresStore) DeleteUser(ctx context.Context, id string) error {
	_, err := r.DB.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, id)
	return err
}

func (r *PostgresStore) GetUserBySessionID(ctx context.Context, sid string) (*User, error) {
	row := r.DB.QueryRowContext(ctx, `
        SELECT u.id, u.email, u.password_hash, u.google_id, u.created_at, u.updated_at
        FROM users u INNER JOIN sessions s ON u.id = s.user_id WHERE s.session_id = $1`, sid)
	u := &User{}
	if err := row.Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.GoogleID, &u.CreatedAt, &u.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return u, nil
}

func (r *PostgresStore) UpdateVerify(ctx context.Context, id string, verify bool) error {
	_, err := r.DB.ExecContext(ctx, `UPDATE users SET verify = $1 WHERE id = $2`, verify, id)
	return err
}
