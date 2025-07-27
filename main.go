package main

import (
	"fmt"
	"log"
	"os"

	"github.com/fuzumoe/linkTorch-api/internal/app"
)

var run = app.Run
var exitFunc = os.Exit

// @title           URL Insight API
// @version         1.0

// @host      localhost:8090
// @BasePath  /api/v1

// @securityDefinitions.basic BasicAuth
// @description Basic Authentication with username and password

// @securityDefinitions.apikey JWTAuth
// @in header
// @name Authorization
// @description JWT Authentication token, prefixed with "Bearer " followed by the token
func main() {
	if err := run(); err != nil {
		log.Printf("error: %v\n", err)
		exitFunc(1)
	}
	fmt.Println("Server shut down cleanly")
}
