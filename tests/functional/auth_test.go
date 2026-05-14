package functional

import (
	"encoding/json"
	"testing"
)

type authResp struct {
	Email string `json:"email"`
	Token string `json:"token"`
}

func TestAuth_RegisterAndLogin(t *testing.T) {
	email := "test@example.com"
	password := "password123"

	// register
	resp := doRequest(t, "POST", "/auth/register", map[string]string{
		"email": email,
		"password": password,
	}, "")
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("register failed: %d", resp.StatusCode)
	}

	var reg authResp
	json.NewDecoder(resp.Body).Decode(&reg)

	if reg.Token == "" {
		t.Fatal("empty token on register")
	}

	// login
	resp = doRequest(t, "POST", "/auth/login", map[string]string{
		"email": email,
		"password": password,
	}, "")
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("login failed: %d", resp.StatusCode)
	}

	var login authResp
	json.NewDecoder(resp.Body).Decode(&login)

	if login.Token == "" {
		t.Fatal("empty token on login")
	}
}