// route registration
package server

import (
	"github.com/labstack/echo/v4"
)

// RegisterRoutes registers all the routes for this api server.

func (s *Server) RegisterRoutes(e *echo.Echo) {
	e.POST("/users", s.RegisterAccount)
	e.POST("/auth/login", s.LoginAccount)

	authenticated := e.Group("/game")
	authenticated.Use(s.JwtAuthMiddleware)
	authenticated.POST("/matchmaking", s.Matchmaking)
}
