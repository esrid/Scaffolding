package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
	"{{projectName}}/db"
	"{{projectName}}/service"

	"golang.org/x/oauth2"
)

func Home(w http.ResponseWriter, r *http.Request) {
	renderPublic(w, nil, "layout.html", "home.html")
}

func GetRegister(w http.ResponseWriter, r *http.Request) {
	renderPublic(w, nil, "layout.html", "register.html")
}

func GetLogin(w http.ResponseWriter, r *http.Request) {
	renderPublic(w, nil, "layout.html", "login.html")
}

func RegisterUser(store db.AuthStore, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		email, password, confirmPassword := r.FormValue("email"), r.FormValue("password"), r.FormValue("confirm_password")
		tolowerall(&email)

		if password != confirmPassword {
			http.Error(w, "Passwords do not match", http.StatusBadRequest)
			return
		}

		created, err := service.RegisterUser(ctx, store, db.User{
			Email:        email,
			PasswordHash: password,
		})
		if err != nil {
			switch err {
			case service.ErrInvalidEmailFormat, service.ErrPasswordTooWeak:
				w.WriteHeader(http.StatusBadRequest)
			case service.ErrEmailAlreadyInUse:
				w.WriteHeader(http.StatusConflict)
			default:
				logger.Error("unable to create user", slog.String("error", err.Error()))
				internal(w)
			}
			return
		}

		cookieHash, err := service.CreateSession(ctx, store, created.ID, r)
		if err != nil {
			logger.Error("unable to create session", slog.String("error", err.Error()))
			internal(w)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "session",
			Value:    cookieHash,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
			MaxAge:   int(24 * time.Hour.Seconds()),
		})

		http.Redirect(w, r, "/app", http.StatusSeeOther)
	}
}

func PostLogin(store db.AuthStore, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		email, password := r.FormValue("email"), r.FormValue("password")
		tolowerall(&email)

		u, err := service.LoginUser(ctx, store, db.User{Email: email, PasswordHash: password})
		switch err {
		case nil:
			cookieHash, err := service.CreateSession(ctx, store, u.ID, r)
			if err != nil {
				internal(w)
			}
			http.SetCookie(w, &http.Cookie{
				Name:     "session",
				Value:    cookieHash,
				Path:     "/",
				HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
				MaxAge:   int(24 * time.Hour.Seconds()),
			})
		case service.ErrInvalidEmailFormat, service.ErrPasswordTooWeak:
			unprocessable(w)
		case sql.ErrNoRows, service.ErrInvalidCredentials:
			unauthorized(w)
		default:
			logger.Error("unable to create user", slog.String("error", err.Error()))
			internal(w)
		}
	}
}

func HandleGoogleLogin(oauth *oauth2.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state, _ := service.GenerateSessionToken()
		http.SetCookie(w, &http.Cookie{
			Name:     "oauth_state",
			Value:    state,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   int(10 * time.Minute.Seconds()),
		})

		url := oauth.AuthCodeURL(state)
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)
	}
}

func HandleGoogleCallback(store db.Store, oauth *oauth2.Config, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var err error

		stateCookie, err := r.Cookie("oauth_state")
		if err != nil || r.URL.Query().Get("state") != stateCookie.Value {
			http.Error(w, "Invalid state parameter", http.StatusBadRequest)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "oauth_state",
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			MaxAge:   -1,
			SameSite: http.SameSiteStrictMode,
		})

		code := r.URL.Query().Get("code")
		token, err := oauth.Exchange(ctx, code)
		if err != nil {
			logger.Error("unable to Exchange code", slog.String("error", err.Error()))
			internal(w)
			return
		}

		client := oauth.Client(ctx, token)
		resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
		if err != nil {
			logger.Error("unable to get the client responses", slog.String("error", err.Error()))
			internal(w)
			return
		}

		defer resp.Body.Close()
		var userInfo struct {
			ID            string `json:"id"`
			Email         string `json:"email"`
			VerifiedEmail bool   `json:"verified_email"`
			Name          string `json:"name"`
			Picture       string `json:"picture"`
		}

		if err = json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
			internal(w)
			return
		}
		if !userInfo.VerifiedEmail {
			http.Error(w, "Email not verified by Google", http.StatusUnauthorized)
			return
		}
		var user *db.User
		user, err = store.GetUserByEmail(ctx, userInfo.Email)
		switch err {
		case nil:
			// User found, continue
		case sql.ErrNoRows:
			newUser := &db.User{
				Email:    userInfo.Email,
				GoogleID: userInfo.ID,
				Oauth:    true,
			}
			user, err = store.CreateUserWithGoogle(ctx, newUser)
			if err != nil {
				logger.Error("unable to create user with google", slog.String("error", err.Error()))
				internal(w)
				return
			}
		default:
			internal(w)
			logger.Error("before session", slog.String("error", err.Error()))
			return
		}

		sessionToken, err := service.CreateSession(ctx, store, user.ID, r)
		if err != nil {
			logger.Error("unable to create session", slog.String("error", err.Error()))
			internal(w)
			return
		}

		csrfToken, err := service.GenerateSessionToken()
		if err != nil {
			logger.Error("unable to genereate session", slog.String("error", err.Error()))
			internal(w)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "session",
			Value:    sessionToken,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
			MaxAge:   int(24 * time.Hour.Seconds()),
		})

		http.SetCookie(w, &http.Cookie{
			Name:     "csrf_token",
			Value:    csrfToken,
			Path:     "/",
			HttpOnly: false,
			SameSite: http.SameSiteStrictMode,
			MaxAge:   int(24 * time.Hour.Seconds()),
		})
	}
}

func GetAdminLogin(w http.ResponseWriter, r *http.Request) {
	renderPublic(w, nil, "layout.html", "admin-login.html")
}

func PostAdminLogin(store db.AuthStore, logger *slog.Logger, mail string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		email, password := r.FormValue("email"), r.FormValue("password")
		tolowerall(&email)

		u, err := service.LoginUser(ctx, store, db.User{Email: email, PasswordHash: password})
		switch err {
		case nil:
			cookieHash, err2 := service.CreateSession(ctx, store, u.ID, r)
			if err2 != nil {
				internal(w)
			}
			http.SetCookie(w, &http.Cookie{
				Name:     "session",
				Value:    cookieHash,
				Path:     "/",
				HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
				MaxAge:   int(24 * time.Hour.Seconds()),
			})

			if err = service.CreateOTP(r.Context(), store, u.ID, mail); err != nil {
				logger.Debug("unable to created or send otp", slog.String("error", err.Error()))
				return
			}

			http.Redirect(w, r, "/admin/verify", http.StatusSeeOther)

		case service.ErrInvalidEmailFormat, service.ErrPasswordTooWeak:
			unprocessable(w)
		case sql.ErrNoRows, service.ErrInvalidCredentials:
			unauthorized(w)
		default:
			logger.Error("unable to create user", slog.String("error", err.Error()))
			internal(w)
		}
	}
}

func GetVerifyOTP(store db.AuthStore, logger *slog.Logger) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := contextUser(r)
		if err := store.UpdateVerify(r.Context(), u.ID, false); err != nil {
			fmt.Printf("err.Error(): %v\n", err.Error())
			internal(w)
			return
		}
		renderPrivate(w, nil, "layout.html", "otp.html")
	})
}

func PostVerifyOTP(store db.AuthStore) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := r.FormValue("code")
		n := len(c)
		if c == "" || n != 6 {
			unprocessable(w)
			return
		}
		u := contextUser(r)

		err := service.ValidateOTP(r.Context(), u.ID, c, store)
		switch err {
		case nil:
			if err := store.UpdateVerify(r.Context(), u.ID, true); err != nil {
				internal(w)
				return
			}
			http.Redirect(w, r, "/admin/dashboard", http.StatusSeeOther)
		case service.ErrInvalidOTPCode:
			unprocessable(w)
		case service.ErrOTPExpired:
			w.Write([]byte("expired code"))
		default:
			internal(w)
		}
	})
}
