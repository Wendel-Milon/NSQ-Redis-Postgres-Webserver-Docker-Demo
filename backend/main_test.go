package main

import (
	"net/http"
	"testing"
	"time"

	"github.com/rs/zerolog/log"
)

func validResponse(res *http.Response, statusCode int, appType string) bool {
	if res.StatusCode != statusCode {
		return false
	}

	ct := res.Header.Get("Content-Type")

	if ct != appType {
		log.Printf("%s, %s", ct, appType)
		return false
	}
	return true
}

func TestFrontPage(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://localhost:8080", nil)

	client := http.Client{Timeout: time.Millisecond * 500}

	for i := 0; i < 5; i++ {
		t.Run("balllern", func(t *testing.T) {
			res, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}

			if !validResponse(res, 200, "text/html; charset=utf-8") {
				t.Fatal("response not Valid!")
			}
		})
	}
}
