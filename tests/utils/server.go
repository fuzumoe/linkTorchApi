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

// StartServer launches the real Gin server exactly once.
func StartServer() {
	startOnce.Do(func() {
		// 0) Load .env so configs.Load sees all variables
		_ = godotenv.Load("../../.env") // silent if missing

		// 1) Force debug unless caller already set it
		if os.Getenv("GIN_MODE") == "" {
			os.Setenv("GIN_MODE", "debug")
		}

		// 2) Pick a test port
		port := os.Getenv("TEST_PORT")
		if port == "" {
			port = "8091"
		}

		// 3) ✨  Tell config loader which port to use
		// (configs.Load reads SERVER_PORT)
		os.Setenv("SERVER_PORT", port)

		// NewClient/BaseURL still rely on PORT →
		os.Setenv("PORT", port)

		// 4) Spin up the server
		go func() {
			if err := app.Run(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("server exited: %v", err)
			}
		}()

		// 5) Block until the port is reachable (max 5 s)
		if err := waitForPort(port, 5*time.Second); err != nil {
			log.Fatalf("server never became ready: %v", err)
		}

		log.Printf("[e2e] test server listening on :%s", port)
	})
}

// waitForPort polls until TCP connect succeeds or times out.
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
