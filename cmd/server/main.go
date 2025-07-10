package main

import (
	"fmt"
	"log"
	"os"

	"github.com/fuzumoe/urlinsight-backend/internal/app"
)

// run is a variable so it can be overridden in tests.
var run = app.Run

// exitFunc is a variable wrapping os.Exit so it can be overridden in tests.
var exitFunc = os.Exit

func main() {
	if err := run(); err != nil {
		log.Fatalf("%v", err)
		// In practice log.Fatalf calls os.Exit, but if you need to call exitFunc:
		exitFunc(1)
	}
	fmt.Println("🎉 Your Gin app is scaffolded. Fill in main.go!")
}
