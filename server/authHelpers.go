package server

import (
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func (s Server) newApiKey(username string) string {
	const expiry = time.Hour * 24 * 30

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		ExpiresAt: &jwt.NumericDate{Time: time.Now().Add(expiry)},
		ID:        username,
	})
	signedToken, err := token.SignedString(s.JwtSecret)
	if err != nil {
		log.Panic("unable to sign api key jwt", err)
	}
	return signedToken
}

// check if api key has expired, and return username of the owner
func (s Server) verifyApiKey(key string) (username string, ok bool) {
	token, err := jwt.ParseWithClaims(key, &jwt.RegisteredClaims{}, func(t *jwt.Token) (any, error) {
		return s.JwtSecret, nil
	})
	if err != nil {
		return "", false
	}
	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok {
		panic("unable to cast to RegisteredClaims. Signature changed")
	}
	// expired?
	if time.Since(claims.ExpiresAt.Time) > 0 {
		return "", false
	}
	return claims.ID, true
}
