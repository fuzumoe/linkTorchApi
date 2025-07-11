package main

import (
	"fmt"
	"log"
	"os"

	"github.com/fuzumoe/urlinsight-backend/internal/app"
)

var run = app.Run
var exitFunc = os.Exit

func main() {
	if err := run(); err != nil {
		log.Printf("error: %v\n", err)
		exitFunc(1)
	}
	fmt.Println("Server shut down cleanly")
}
