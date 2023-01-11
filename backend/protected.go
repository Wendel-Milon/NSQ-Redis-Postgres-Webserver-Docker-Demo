package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
)

func (server *Server) ValidateSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		cookie, err := r.Cookie("csrftoken")
		if err != nil {
			fmt.Println("Middleware Validate caught csrf-token not set", err)
			http.Redirect(w, r, "/login", http.StatusUnauthorized)
			return
		}

		_, err = server.redis.Get(context.Background(), cookie.Value).Result()
		if err != nil {
			fmt.Println("Middleware Validate caught csrf-token does not exist", err)
			http.Redirect(w, r, "/login", http.StatusUnauthorized)
			return
		}

		// fmt.Println("Middleware called", cookie.Value)
		next.ServeHTTP(w, r)
	})
}

func (server *Server) ProduceToNSQGET(w http.ResponseWriter, r *http.Request) {

	// The iframe is there so that you will NOT be redirected to a new page.
	html := `
		<h1>Protected Success</h1>
		<p>This page can only be reached when a valid crsf-token is set.</p>
		
		<iframe name="dummyframe" id="dummyframe" style="display: none;"></iframe>
		<form action="/protected" method="post" target="dummyframe">
			<input type="submit" name="NSQmessage" value="Produce NSQ Message" />
		</form>
		`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Add("Random", "text/hmtl; charset=utf-8")

	w.WriteHeader(http.StatusOK)

	_, err := w.Write([]byte(html))

	if err != nil {
		server.SendError(w, r)
		return
	}
}

func (server *Server) ProduceToNSQPOST(w http.ResponseWriter, r *http.Request) {

	//TODO enable selection of topic and message
	message := "default message"

	err := server.nsq.Publish("default", []byte(message))
	if err != nil {
		server.SendErrorMessage(w, r, http.StatusBadRequest, err.Error())
		log.Println("Error when producing message", err)
		return
	}
	fmt.Println("Succesfully produced message")
}