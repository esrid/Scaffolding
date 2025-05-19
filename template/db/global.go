package db

type Store interface {
	SessionStore
	UserStore
	OtpStore
}
