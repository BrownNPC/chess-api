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
type ApiKeyResponse struct {
	ApiKey string `json:"apiKey"`
}

// Create a user account using provided username and password.
//
//	@Summary		Create an account using provided username and password.
//	@Description	Username can be between 3-20 characters.
//	@Description	Password must be at least 3 characters.
//
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		UserCredentials	true	"Register Account"
//	@Success		201		{object}	ApiKeyResponse	"Api Key"
//	@Failure		400		{object}	ErrorReason		"Invalid credentials"
//	@Failure		409		{object}	ErrorReason		"Username already exists"
//	@Failure		500		{object}	ErrorReason
//	@Router			/users [post]
func (s Server) RegisterUserAccount(c echo.Context) error {
	var req UserCredentials
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, REASON_JSON_SYNTAX_ERROR)
	}
	// validate username and password
	if err := ValidateUsernameAndPassword(req.Username, req.Password); err != nil {
		return c.JSON(http.StatusBadRequest, Reason(err.Error()))
	}

	// check if username already exists
	user, _ := s.DB.GetUserByUsername(c.Request().Context(), req.Username)
	if user.Username != "" {
		return c.JSON(http.StatusConflict, Reason("Username already exists"))
	}
	// generate password hash
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("Failed to hash password", "password", req.Password, "error", err)
		return c.JSON(http.StatusInternalServerError, REASON_INTERNAL_ERROR)
	}
	// create user in the database
	user, err = s.DB.CreateUser(c.Request().Context(), db.CreateUserParams{
		Username:     req.Username,
		PasswordHash: string(passwordHash),
		ApiKey:       s.newApiKey(req.Username),
	})

	if err != nil {
		slog.Error("failed to create user, guard statements should stop this", "error", err)
		return c.JSON(http.StatusInternalServerError, REASON_INTERNAL_ERROR)
	}

	return c.JSON(http.StatusCreated, ApiKeyResponse{user.ApiKey})
}

// @Summary	Delete an account
//
// @Tags		users
// @Accept		json
// @Produce	json
// @Param		Authorization	header		string	true	"Must contain ApiKey in the format Bearer: apiKey"
// @Success	200				{object}	string	"deleted"
// @Failure	401				{object}	ErrorReason
// @Failure	500				{object}	ErrorReason
// @Router		/users [delete]
func (s Server) DeleteUserAccount(c echo.Context) error {
	username := c.Get("username").(string)
	if username == "" {
		return c.JSON(http.StatusUnauthorized, REASON_UNAUTHORIZED)
	}
	user, err := s.DB.GetUserByUsername(c.Request().Context(), username)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, REASON_INTERNAL_ERROR)
	}
	err = s.DB.DeleteUser(c.Request().Context(), user.Uid)
	if err != nil {
		slog.Warn("user exists in DB but we cannot delete it", "username", username)
		return c.JSON(http.StatusInternalServerError, REASON_INTERNAL_ERROR)
	}

	return c.JSON(http.StatusOK, "deleted")
}
