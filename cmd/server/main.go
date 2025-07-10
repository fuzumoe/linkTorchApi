// main.go
package main

import (
	"fmt"
	"log"

	"github.com/fuzumoe/urlinsight-backend/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatalf("%v", err)
	}
	fmt.Println("🎉 Your Gin app is scaffolded. Fill in main.go!")
}
