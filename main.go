//go:generate go run github.com/swaggo/swag/cmd/swag@latest init
package main

import (
	"api/server"
	"context"
	"database/sql"
	_ "embed"
	"log"

	_ "api/docs"

	"github.com/labstack/echo/v4"
	_ "modernc.org/sqlite"

	echoSwagger "github.com/swaggo/echo-swagger"
)

//go:embed schema.sql
var DATABASE_SCHEMA string

//	@title			Chess API
//	@description	chess api for playing chess online.

// @license.name	MIT
func main() {
	ctx := context.Background()
	dbconn, err := sql.Open("sqlite", "sqlite.db")
	if err != nil {
		log.Fatal(err)
	}
	defer dbconn.Close()

	// create tables if not present
	dbconn.ExecContext(ctx, DATABASE_SCHEMA)

	e := echo.New()

	srv := server.NewServer(dbconn)

	e.GET("/", func(c echo.Context) error {
		return c.Redirect(302, "/swagger/index.html")
	})
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	srv.RegisterRoutes(e)

	err = e.Start(":8080")
	if err != nil {
		log.Fatal("Server shutdown", err)
	}
}
