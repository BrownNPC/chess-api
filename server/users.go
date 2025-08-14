// All routes related to users.
package server

import (
	"api/db"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

type RegisterAccountRequest struct {
	Username string
	Password string
}

// Create an account using provided username and password.
// respond String if error, status created + no body if success
//
//	@Summary		Create an account
//	@Description	Create an account using provided username and password
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			payload	body	RegisterAccountRequest	true	"Register Account"
//	@Success		201
//	@Failure		400	{string}	string	"Malformed credentials"
//	@Failure		409	{string}	string	"Username already exists"
//	@Failure		500
//
//	@Router			/users [post]
func (s *Server) RegisterAccount(c echo.Context) error {
	var req RegisterAccountRequest

	if err := c.Bind(&req); err != nil {
		return c.String(http.StatusBadRequest, "Json body contains syntax error")
	}
	if err := ValidateUsername(req.Username); err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	if err := ValidatePassword(req.Password); err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	// check if username already exists
	user, err := s.DB.GetUserByUsername(c.Request().Context(), req.Username)
	if user.Username != "" {
		return c.String(http.StatusConflict, "Username already taken")
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("Failed to hash password", "password", req.Password, "error", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	_, err = s.DB.CreateUser(c.Request().Context(), db.CreateUserParams{
		Username:     req.Username,
		PasswordHash: string(passwordHash),
	})
	if err != nil {
		slog.Error("failed to create user, guard statements should stop this", "error", err)
	}

	return c.NoContent(http.StatusCreated)
}

var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]*$`)

func ValidatePassword(password string) error {
	if len([]rune(password)) < 3 {
		return fmt.Errorf("password must be at least 3 characters")
	}
	return nil
}

const INVALID_USERNAME_ERROR = "username can only contain letters, numbers, and underscores"

func ValidateUsername(username string) error {
	length := len([]rune(username))
	if length < 3 {
		return errors.New("username must be at least 3 characters long")
	}
	if length > 20 {
		return errors.New("username cannot be longer than 20 characters")
	}
	if !usernameRegex.MatchString(username) {
		return errors.New(INVALID_USERNAME_ERROR)
	}
	return nil
}
