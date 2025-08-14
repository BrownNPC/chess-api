// All routes related to users.
package server

import (
	"api/db"
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

// User is the representation of a user account that will be returned by the api
type User struct {
	UserID    int64     `json:"userId" example:"12"`
	Username  string    `json:"username" example:"JohnDoe"`
	CreatedAt time.Time `json:"createdAt" format:"date-time"`
}

// UserCredentials are the required credentials to make a an account and log in.
type UserCredentials struct {
	Username string `json:"username" minLength:"4" maxLength:"20" example:"JohnDoe"`
	Password string `json:"password" minLength:"3" example:"Password123"`
}

// Create an account using provided username and password.
// respond String if error, status created + no body if success
//
//	@Summary		Create an account
//	@Description	Create an account using provided username and password.
//	@Description	Username can be between 3-20 characters.
//	@Description	Password must ba at least 3 characters.
//
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		UserCredentials	true	"Register Account"
//	@Success		201		{object}	User
//	@Failure		400		{object}	ErrorReason	"Unallowed credentials"
//	@Failure		409		{object}	ErrorReason	"Username already exists"
//	@Failure		500		{object}	ErrorReason
//	@Router			/users [post]
func (s Server) RegisterAccount(c echo.Context) error {
	var req UserCredentials
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, Reason("Json body contains syntax error"))
	}
	// validate username
	if err := ValidateUsername(req.Username); err != nil {
		return c.JSON(http.StatusBadRequest, Reason(err.Error()))
	}
	// validate password
	if err := ValidatePassword(req.Password); err != nil {
		return c.JSON(http.StatusBadRequest, Reason(err.Error()))
	}
	// check if username already exists
	user, err := s.DB.GetUserByUsername(c.Request().Context(), req.Username)
	if user.Username != "" {
		return c.JSON(http.StatusConflict, Reason("Username already taken"))
	}
	// generate password hash
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("Failed to hash password", "password", req.Password, "error", err)
		return c.JSON(http.StatusInternalServerError, Reason("internal server error"))
	}
	// create user in the database
	user, err = s.DB.CreateUser(c.Request().Context(), db.CreateUserParams{
		Username:     req.Username,
		PasswordHash: string(passwordHash),
	})
	if err != nil {
		slog.Error("failed to create user, guard statements should stop this", "error", err)
		return c.JSON(http.StatusInternalServerError, Reason("internal server error"))
	}

	return c.JSON(http.StatusCreated, User{
		Username:  user.Username,
		UserID:    user.Uid,
		CreatedAt: user.CreatedAt,
	})
}

func (s Server) LoginAccount(c echo.Context) error {
	return nil
}
