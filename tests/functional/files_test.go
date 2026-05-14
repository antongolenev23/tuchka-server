package functional

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"testing"
)

func authToken(t *testing.T) string {
	resp := doRequest(t, "POST", "/auth/login", map[string]string{
		"email": "test@example.com",
		"password": "password123",
	}, "")
	defer resp.Body.Close()

	var r struct {
		Token string `json:"token"`
	}

	json.NewDecoder(resp.Body).Decode(&r)

	return "Bearer " + r.Token
}

func TestFiles_UploadListDelete(t *testing.T) {
	token := authToken(t)

	// upload
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("files", "test.txt")
	if err != nil {
		t.Fatal(err)
	}

	part.Write([]byte("hello world"))
	writer.Close()

	req, err := http.NewRequest("POST", baseURL+"/files/upload", &buf)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", token)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("upload failed: %d", resp.StatusCode)
	}

	// list
	resp = doRequest(t, "GET", "/files", nil, token)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("list failed: %d", resp.StatusCode)
	}

	var files []map[string]any
	json.NewDecoder(resp.Body).Decode(&files)

	if len(files) == 0 {
		t.Fatal("no files found")
	}

	// delete
	del := map[string][]string{
		"files": {files[0]["name"].(string)},
	}

	resp = doRequest(t, "POST", "/files/delete", del, token)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("delete failed: %d", resp.StatusCode)
	}
}