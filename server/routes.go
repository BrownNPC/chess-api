// route registration
package server

import (
	"github.com/labstack/echo/v4"
)

// RegisterRoutes registers all the routes for this api server.

func (s *Server) RegisterRoutes(e *echo.Echo) {

	e.POST("/users", s.RegisterUserAccount)

	e.POST("/matches", s.CreateMatch, s.AuthApiKeyMiddleware)
	e.GET("/matches/:id/play", s.JoinMatch, s.AuthApiKeyMiddleware)
	e.PUT("/matches/:id", s.PutMove, s.AuthApiKeyMiddleware)
	e.GET("/matches/:id", s.GetBoardString)
	e.GET("/matches/:id/img", s.GetBoardImage, s.AuthApiKeyMiddleware)

	e.POST("/auth/login", s.GetApiKeyTryRenew)
}
