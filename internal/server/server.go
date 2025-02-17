package server

import (
	"github.com/gofiber/fiber/v2"

	"Backend-Bluelock-007/internal/database"
)

type FiberServer struct {
	*fiber.App

	db database.Service
}

func New() *FiberServer {
	server := &FiberServer{
		App: fiber.New(fiber.Config{
			ServerHeader: "Backend-Bluelock-007",
			AppName:      "Backend-Bluelock-007",
		}),

		db: database.New(),
	}

	return server
}
