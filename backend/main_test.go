package main

import (
	"net/http"
	"testing"
	"time"
)

func TestFrontPage(t *testing.T) {
	req, _ := http.NewRequest("GET", "http:localhost:8080", nil)

	client := http.Client{Timeout: time.Millisecond * 500}

	for i := 0; i < 1000000; i++ {
		go t.Run("balllern", func(t *testing.T) {
			_, err := client.Do(req)
			if err != nil {
				t.Fail()
			}
		})

	}
}
