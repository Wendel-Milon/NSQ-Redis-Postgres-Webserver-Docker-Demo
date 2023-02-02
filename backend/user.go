package main

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func (server *Server) CreateUserGET(w http.ResponseWriter, r *http.Request) {
	html := `
		<h1>Create User</h1>
		<form action="/create" method="post">
			<label for="userid">User ID:</label><br>
			<input type="text" id="userid" name="userid"><br>
			<label for="passwd">Password:</label><br>
			<input type="text" id="passwd" name="passwd">
			<input type="submit" value="Create">
	  	</form>
	  `
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Add("Random", "text/hmtl; charset=utf-8")

	w.WriteHeader(http.StatusOK)

	_, err := w.Write([]byte(html))

	if err != nil {
		log.Warn().Err(err).Caller().Msg("")
		server.SendError(w, r)
		return
	}
}

func (server *Server) CreateUserPOST(w http.ResponseWriter, r *http.Request) {
	// if r.Method == "POST" {

	err := r.ParseForm()
	if err != nil {
		server.SendErrorMessage(w, r, http.StatusBadRequest, err.Error())
		log.Warn().Err(err).Caller().Msg("CreateUserPOST")
		return
	}

	userid, ok := r.Form["userid"]
	if !ok {
		server.SendErrorMessage(w, r, http.StatusBadRequest, ErrNoUserID.Error()) // Maybe just index array....
		log.Warn().Err(err).Caller().Msg("CreateUserPOST")
		return
	}
	joinedUser := strings.Join(userid, "")

	passwd, ok := r.Form["passwd"]
	if !ok {
		log.Warn().Err(err).Caller().Msg("CreateUserPOST")
		server.SendErrorMessage(w, r, http.StatusBadRequest, ErrNoPassWd.Error())
		return
	}
	joined := strings.Join(passwd, "") // Maybe just index array....

	//TODO Check for userId already exits!.

	hash, err := bcrypt.GenerateFromPassword([]byte(joined), bcrypt.DefaultCost)
	if err != nil {
		log.Warn().Err(err).Caller().Msg("CreateUserPOST")
		server.SendErrorMessage(w, r, http.StatusBadRequest, err.Error())
		return
	}

	sql := `INSERT INTO users (userid, passwd) VALUES ($1, $2)`

	_, err = server.pg.Exec(context.Background(), sql, joinedUser, hash)
	if err != nil {
		log.Warn().Err(err).Caller().Msg("CreateUserPOST")
		server.SendErrorMessage(w, r, http.StatusBadRequest, err.Error())
		return
	}

	log.Info().Msgf("User successfully created", userid)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (server *Server) LoginUserGET(w http.ResponseWriter, r *http.Request) {
	html := `
		<h1>Login</h1>
		<form action="/login" method="post">
			<label for="userid">User ID:</label><br>
			<input type="text" id="userid" name="userid"><br>
			<label for="passwd">Password:</label><br>
			<input type="text" id="passwd" name="passwd">
			<input type="submit" value="Create">
	  	</form>
	  `
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Add("Random", "text/hmtl; charset=utf-8")

	w.WriteHeader(http.StatusOK)

	_, err := w.Write([]byte(html))

	if err != nil {
		log.Warn().Err(err).Caller().Msg("LoginUserGET")
		server.SendError(w, r)
		return
	}
}

func (server *Server) LoginUserPOST(w http.ResponseWriter, r *http.Request) {

	err := r.ParseForm()
	if err != nil {
		log.Warn().Err(err).Caller().Msg("LoginUserGET")
		server.SendErrorMessage(w, r, http.StatusBadRequest, err.Error())
		return
	}

	userid, ok := r.Form["userid"]
	if !ok {
		log.Warn().Err(err).Caller().Msg("LoginUserGET")
		server.SendErrorMessage(w, r, http.StatusBadRequest, "notOK") // Maybe just index array....
		return
	}
	joinedUser := strings.Join(userid, "")

	passwd, ok := r.Form["passwd"]
	if !ok {
		log.Warn().Err(err).Caller().Msg("LoginUserGET")
		server.SendErrorMessage(w, r, http.StatusBadRequest, "notOK")
		return
	}
	joined := strings.Join(passwd, "") // Maybe just index array....

	var pwhash []byte
	err = server.pg.QueryRow(context.Background(), "select passwd FROM users where userid=$1", joinedUser).Scan(&pwhash)
	if err != nil {
		log.Warn().Err(err).Caller().Msg("LoginUserGET")
		server.SendErrorMessage(w, r, http.StatusBadRequest, err.Error())
		return
	}

	err = bcrypt.CompareHashAndPassword(pwhash, []byte(joined))
	if err != nil {
		log.Warn().Err(err).Caller().Msg("LoginUserGET")
		server.SendErrorMessage(w, r, http.StatusBadRequest, err.Error())
		return
	}

	// Create Cookie
	token, err := uuid.NewRandom()
	if err != nil {
		log.Warn().Err(err).Caller().Msg("LoginUserGET")
		server.SendErrorMessage(w, r, http.StatusBadRequest, err.Error())
		return
	}
	// Add to Redis
	err = server.redis.Set(context.Background(), token.String(), token.String(), time.Minute*10).Err()
	if err != nil {
		log.Warn().Err(err).Caller().Msg("LoginUserGET")
		server.SendErrorMessage(w, r, http.StatusBadRequest, err.Error())
		return
	}

	// Return Cookie
	cookie := http.Cookie{
		Name:  "csrftoken",
		Value: token.String(),
		// RawExpires: ,
	}

	http.SetCookie(w, &cookie)

	log.Info().Msgf("User", joinedUser, "logged into System.")
	http.Redirect(w, r, "/protected", http.StatusSeeOther)
}

func (server *Server) LogoutUserPOST(w http.ResponseWriter, r *http.Request) {

	cookie, err := r.Cookie("csrftoken")
	if err != nil {
		log.Warn().Err(err).Caller().Msg("LogoutUserPOST")
		// log.Info().Msgf("No Cookie was set. Request was useless!", err)
		return
	}

	i, err := server.redis.Del(context.Background(), cookie.Value).Result()
	if err != nil {
		log.Warn().Err(err).Caller().Msg("LogoutUserPOST")
		// http.Redirect(w, r, "/login", http.StatusUnauthorized)
		return
	}
	log.Info().Msgf("Deletion of the cookie was successful!", cookie, i)
	// Return Cookie
	newCookie := http.Cookie{
		Name:  "csrftoken",
		Value: "",
		// RawExpires: ,
	}

	http.SetCookie(w, &newCookie)
}
