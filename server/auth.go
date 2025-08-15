package server

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

// JwtClaims defines custom JWT claims along with registered claims.
type JwtClaims struct {
	User
	jwt.RegisteredClaims
}

func (s Server) JwtAuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// extract Authorization: Bearer <token>
		ah := c.Request().Header.Get(echo.HeaderAuthorization) // case-insensitive
		if ah == "" {
			return c.JSON(http.StatusForbidden, REASON_INVALID_AUTH_HEADER)
		}

		bearerJwt := strings.Split(ah, " ")
		if len(bearerJwt) != 2 {
			return c.JSON(http.StatusForbidden, REASON_INVALID_AUTH_HEADER)
		}
		// Bearer xxxx.yyyy.zzzz
		// get rid of the "Bearer "
		encodedToken := bearerJwt[1]
		// parse encoded token
		token, err := jwt.ParseWithClaims(encodedToken, &JwtClaims{}, func(t *jwt.Token) (any, error) {
			return s.JwtSecret, nil
		})
		// failed to parse?
		if err != nil {
			slog.Error("failed to parse jwt", "error", err)
			return c.JSON(http.StatusForbidden, REASON_INVALID_AUTH_HEADER)
		}
		if claims, ok := token.Claims.(*JwtClaims); ok {
			// valid token, continue
			c.Set("user", claims)
			return next(c)
		} else {
			panic("Failed to decode jwt into struct. This means the jwt we are sending is wrong")
		}
	}
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
//	@Tags			auth
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
