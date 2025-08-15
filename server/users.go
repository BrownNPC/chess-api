// All routes related to users.
package server

import (
	"api/db"
	"log/slog"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
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
//	@Failure		400		{object}	ErrorReason	"Invalid credentials"
//	@Failure		409		{object}	ErrorReason	"Username already exists"
//	@Failure		500		{object}	ErrorReason
//	@Router			/users [post]
func (s Server) RegisterAccount(c echo.Context) error {
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
	})
	if err != nil {
		slog.Error("failed to create user, guard statements should stop this", "error", err)
		return c.JSON(http.StatusInternalServerError, REASON_INTERNAL_ERROR)
	}

	return c.JSON(http.StatusCreated, UserFromDbUser(user))
}

// JwtClaims defines custom JWT claims along with registered claims.
type JwtClaims struct {
	User
	jwt.RegisteredClaims
}
type JwtResponse struct {
	JWT string `json:"jwt" example:"xxxx.yyyy.zzzz"`
}

// LoginAccount accepts username and password, and returns a JWT
//
//	@Summary		Log into an account and get a JWT
//	@Description	Log into an account using provided username and password. And get a JWT
//	@Description	Username can be between 3-20 characters.
//	@Description	Password must be at least 3 characters.
//
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		UserCredentials	true	"Login Account"
//	@Success		201		{object}	JwtResponse
//	@Failure		401		{object}	ErrorReason	"Invalid username/password"
//	@Failure		500		{object}	ErrorReason
//	@Router			/auth/login [post]
func (s Server) LoginAccount(c echo.Context) error {
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
	// create jwt that expires in 1 day
	claims := JwtClaims{
		User: UserFromDbUser(user),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	// create jwt
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	finalJwt, err := token.SignedString(s.JwtSecret)
	if err != nil {
		slog.Error("Failed to sign jwt", "error", err)
		return c.JSON(http.StatusInternalServerError, REASON_INTERNAL_ERROR)
	}

	return c.JSON(http.StatusOK, JwtResponse{JWT: finalJwt})
}
