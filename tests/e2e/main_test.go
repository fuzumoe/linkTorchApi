package e2e_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"testing"

	"github.com/fuzumoe/linkTorch-api/tests/utils"
)

func TestMain(m *testing.M) {
	// Set test mode to true so that repositories can use silent logging.
	os.Setenv("TEST_MODE", "true")

	// 1) start the server
	utils.StartServer()

	// 2) register TEST_USER with DEV Basic creds
	registerTestUser()

	os.Exit(m.Run())
}

func apiPath(p string) string {
	return "/api/v1" + p
}

func registerTestUser() {
	devEmail := os.Getenv("DEV_USER_EMAIL")
	devPass := os.Getenv("DEV_USER_PASSWORD")
	testEmail := os.Getenv("TEST_USER_EMAIL")
	testName := os.Getenv("TEST_USER_NAME")
	testPass := os.Getenv("TEST_USER_PASSWORD")

	if devEmail == "" || devPass == "" || testEmail == "" || testPass == "" {
		log.Println("missing DEV_USER_* or TEST_USER_* env, skipping test‐user registration")
		return
	}

	payload := map[string]string{
		"email":    testEmail,
		"username": testName,
		"password": testPass,
	}
	body, _ := json.Marshal(payload)

	// call POST /register with Basic Auth = DEV_USER
	c := utils.NewClient()
	req, err := http.NewRequest("POST", apiPath("/register"), bytes.NewReader(body))
	if err != nil {
		log.Printf("registerTestUser: NewRequest error: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	basic := base64.StdEncoding.EncodeToString([]byte(devEmail + ":" + devPass))
	req.Header.Set("Authorization", "Basic "+basic)

	resp, err := c.Do(req)
	if err != nil {
		log.Printf("registerTestUser: request error: %v", err)
		return
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusCreated:
		log.Printf("registerTestUser: created %s", testEmail)
	case http.StatusConflict:
		log.Printf("registerTestUser: %s already exists", testEmail)
	default:
		log.Printf("registerTestUser: unexpected status %d", resp.StatusCode)
	}
}
