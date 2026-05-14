package functional

import (
	"bytes"
	"encoding/json"
	"crypto/tls"
	"net/http"
	"testing"
	"time"
)

var baseURL = "https://127.0.0.1:8443"

var client = &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	},
}

func doRequest(t *testing.T, method, url string, body any, token string) *http.Response {
	var buf bytes.Buffer

	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatal(err)
		}
	}

	req, err := http.NewRequest(method, baseURL+url, &buf)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)
	
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	return resp
}