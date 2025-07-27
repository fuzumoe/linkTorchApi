package utils

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/joho/godotenv"

	"github.com/fuzumoe/linkTorch-api/internal/app"
)

var startOnce sync.Once

func StartServer() {
	startOnce.Do(func() {

		_ = godotenv.Load("../../.env")

		if os.Getenv("GIN_MODE") == "" {
			os.Setenv("GIN_MODE", "debug")
		}

		port := os.Getenv("TEST_PORT")
		if port == "" {
			port = "8091"
		}

		os.Setenv("SERVER_PORT", port)

		os.Setenv("PORT", port)

		go func() {
			if err := app.Run(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("server exited: %v", err)
			}
		}()

		if err := waitForPort(port, 5*time.Second); err != nil {
			log.Fatalf("server never became ready: %v", err)
		}

		log.Printf("[e2e] test server listening on :%s", port)
	})
}

func waitForPort(p string, d time.Duration) error {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", "localhost:"+p, 50*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("timeout after %s", d)
}
