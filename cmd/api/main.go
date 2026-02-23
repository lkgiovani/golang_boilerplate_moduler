package main

import (
	"golang_boilerplate_module/internal/bootstrap"

	"github.com/joho/godotenv"
	"go.uber.org/fx"
)

func main() {

	_ = godotenv.Load()

	app := fx.New(bootstrap.App)
	app.Run()
}
