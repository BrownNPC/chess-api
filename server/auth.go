package server

import (
	"api/db"
	"log/slog"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

// AuthApiKeyMiddleware checks the Authorization header for a Bearer <api key>.
// It sets the context's username field to the username of whom the key belongs to.
// Otherwise, username is an empty string.
func (s Server) AuthApiKeyMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// extract Authorization header
		ah := c.Request().Header.Get(echo.HeaderAuthorization) // case-insensitive
		if ah == "" {
			// unauthorized user, set username to empty string
			c.Set("username", "")
			return next(c)
		}
		// seperate "Bearer" from api key
		bearerJwt := strings.Split(ah, " ")
		if len(bearerJwt) != 2 {
			return c.JSON(http.StatusForbidden, REASON_INVALID_AUTH_HEADER)
		}
		// Bearer xxxx.yyyy.zzzz
		// get rid of the "Bearer "
		encodedToken := bearerJwt[1]
		// parse encoded token
		token, err := jwt.Parse(encodedToken, func(t *jwt.Token) (any, error) {
			return s.JwtSecret, nil
		}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()})) // failed to parse?
		if err != nil {
			return c.JSON(http.StatusUnauthorized, REASON_INVALID_AUTH_HEADER)
		}
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			// valid token, continue
			username := claims["jti"].(string)
			user, err := s.DB.GetUserByUsername(c.Request().Context(), username)
			if err != nil {
				return c.JSON(http.StatusForbidden, Reason("user does not exist"))
			}
			// check if token has expired
			if user.ApiKey != encodedToken {
				return c.JSON(http.StatusForbidden, Reason("Key has expired"))
			}
			_, ok := s.verifyApiKey(encodedToken)
			if !ok {
				return c.JSON(http.StatusForbidden, Reason("Key has expired"))
			}

			c.Set("username", username)
			return next(c)
		} else {
			panic("Failed to decode jwt into struct. This means the jwt we are sending is wrong")
		}
	}
}

// GetApiKeyTryRenew accepts username and password, and returns an api key.
// Accounts can be created from /users
//
//	@Summary		Log into an account and get an API key.
//	@Description	Log into an account using provided username and password. And get an API key.
//	@Description	Username can be between 3-20 characters.
//	@Description	Password must be at least 3 characters.
//
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		UserCredentials	true	"Login Account"
//	@Success		201		{object}	ApiKeyResponse
//	@Failure		401		{object}	ErrorReason	"Invalid username/password"
//	@Failure		500		{object}	ErrorReason
//	@Router			/auth/login [post]
func (s Server) GetApiKeyTryRenew(c echo.Context) error {
	var req UserCredentials

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, REASON_JSON_SYNTAX_ERROR)
	}

	// validate username and password
	if err := ValidateUsernameAndPassword(req.Username, req.Password); err != nil {
		return c.JSON(http.StatusBadRequest, Reason(err.Error()))
	}

	// get user
	user, err := s.DB.GetUserByUsername(c.Request().Context(), req.Username)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, REASON_INVALID_CREDENTIALS)
	}
	// validate password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return c.JSON(http.StatusUnauthorized, REASON_INVALID_CREDENTIALS)
	}
	username, ok := s.verifyApiKey(user.ApiKey)
	if username != user.Username {
		slog.Warn("Username stored in api key does not match user we got from database. This should never happen.")
		return c.JSON(http.StatusInternalServerError, REASON_INTERNAL_ERROR)
	}
	if !ok { // api key expired
		user.ApiKey = s.newApiKey(user.Username)
		err := s.DB.UpdateUserAPIKey(c.Request().Context(), db.UpdateUserAPIKeyParams{
			ApiKey:   user.ApiKey,
			Username: req.Username,
		})
		if err != nil {
			slog.Warn("could not update api key for user", "error", err)
			return c.JSON(http.StatusInternalServerError, REASON_INTERNAL_ERROR)
		}
	}
	return c.JSON(http.StatusOK, ApiKeyResponse{user.ApiKey})
}
