package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"net/http"
	"time"
)

type Claims struct {
	jwt.RegisteredClaims
	UserID int
}

const TOKEN_EXP = time.Hour * 48
const SECRET_KEY = "supersecretkey"
const USERID_KEY string = "userID"

func WithAuthorization(next func(w http.ResponseWriter, r *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("authorization_token")
		if err != nil {
			switch {
			case errors.Is(err, http.ErrNoCookie):
				cookie, err = NewCokie()
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
				http.SetCookie(w, cookie)
			default:
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}
		tokenString := cookie.Value
		userId := GetUserId(tokenString)
		if userId <= -1 {
			cookie, err = NewCokie()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			http.SetCookie(w, cookie)
		}
		if userId == 0 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		}
		ctx := context.WithValue(r.Context(), USERID_KEY, userId)
		r = r.WithContext(ctx)
		next(w, r)
	}

}

func NewCokie() (*http.Cookie, error) {
	tokenString, err := NewAuthToken()
	if err != nil {
		return nil, err
	}
	return &http.Cookie{
		Name:     "authorization_token",
		Value:    tokenString,
		Path:     "/",                  // Доступ на всем сайте
		HttpOnly: true,                 // Защита от XSS
		Secure:   true,                 // Только HTTPS
		SameSite: http.SameSiteLaxMode, // Защита от CSRF
	}, nil
}

func NewAuthToken() (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			// когда создан токен
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TOKEN_EXP)),
		},
		UserID: usersCount + 1,
	})

	// создаём строку токена
	tokenString, err := token.SignedString([]byte(SECRET_KEY))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func GetUserId(tokenString string) int {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(SECRET_KEY), nil
		})
	if err != nil || !token.Valid {
		return -1
	}
	return claims.UserID
}
