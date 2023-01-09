package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-redis/redis/v9"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

var CacheURL = os.Getenv("CACHE_URL")
var PgURL = os.Getenv("DATABASE_URL")

type Server struct {
	redis *redis.Client
	pg    *pgx.Conn
}

var server Server

func SetupServer() error {
	// server := Server{}

	/************************ REDIS ***************************/

	// This does not Check if the client even exits!
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:6379", CacheURL),
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	// Testing for a valid connection.
	err := rdb.Set(context.Background(), "TEST", "Connection", 0).Err()
	if err != nil {
		return fmt.Errorf("unable to set redis value: %v", err)

	}
	err = rdb.Del(context.Background(), "TEST").Err()
	if err != nil {
		return fmt.Errorf("unable to delete the redis key: %v", err)
	}
	server.redis = rdb

	log.Println("Successfully connected to Redis")

	/************************** POSTGRES ***********************/

	conn, err := pgx.Connect(context.Background(),
		PgURL)
	if err != nil {
		return fmt.Errorf("unable to connect to database: %v", err)
	}
	server.pg = conn

	log.Println("Successfully connected to Postgres")

	return nil

}

func SendError(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("sth went wrong"))
}

func SendErrorMessage(w http.ResponseWriter, r *http.Request, message string) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(message))
}

func CreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
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
			SendError(w, r)
			return
		}
		return
	}

	if r.Method == "POST" {

		err := r.ParseForm()
		if err != nil {
			SendErrorMessage(w, r, err.Error())
			return
		}

		userid, ok := r.Form["userid"]
		if !ok {
			SendErrorMessage(w, r, "notOK") // Maybe just index array....
			return
		}
		joinedUser := strings.Join(userid, "")

		passwd, ok := r.Form["passwd"]
		if !ok {
			SendErrorMessage(w, r, "notOK")
			return
		}
		joined := strings.Join(passwd, "") // Maybe just index array....

		//TODO Check for userId already exits!.

		hash, err := bcrypt.GenerateFromPassword([]byte(joined), bcrypt.DefaultCost)
		if err != nil {
			SendErrorMessage(w, r, err.Error())
			return
		}

		sql := `INSERT INTO users (userid, passwd) VALUES ($1, $2)`

		_, err = server.pg.Exec(context.Background(), sql, joinedUser, hash)
		if err != nil {
			SendErrorMessage(w, r, err.Error())
			return
		}

		log.Println("User successfully created", userid)
	}
}

func LoginUser(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
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
			SendError(w, r)
			return
		}
		return
	}

	if r.Method == "POST" {

		err := r.ParseForm()
		if err != nil {
			SendErrorMessage(w, r, err.Error())
			return
		}

		userid, ok := r.Form["userid"]
		if !ok {
			SendErrorMessage(w, r, "notOK") // Maybe just index array....
			return
		}
		joinedUser := strings.Join(userid, "")

		passwd, ok := r.Form["passwd"]
		if !ok {
			SendErrorMessage(w, r, "notOK")
			return
		}
		joined := strings.Join(passwd, "") // Maybe just index array....

		var pwhash []byte
		err = server.pg.QueryRow(context.Background(), "select passwd FROM users where userid=$1", joinedUser).Scan(&pwhash)
		if err != nil {
			SendErrorMessage(w, r, err.Error())
			return
		}

		err = bcrypt.CompareHashAndPassword(pwhash, []byte(joined))
		if err != nil {
			SendErrorMessage(w, r, err.Error())
			return
		}

		// Create Cookie
		token, err := uuid.NewRandom()
		if err != nil {
			SendErrorMessage(w, r, err.Error())
			return
		}
		// Add to Redis
		err = server.redis.Set(context.Background(), token.String(), token, time.Minute).Err()
		if err != nil {
			SendErrorMessage(w, r, err.Error())
			return
		}

		// Return Cookie
		cookie := http.Cookie{
			Name:  "csrftoken",
			Value: token.String(),
			// RawExpires: ,
		}

		http.SetCookie(w, &cookie)

		log.Println("User", joinedUser, "logged into System.")
		http.Redirect(w, r, "/protected", http.StatusSeeOther)
		return
	}
}

func ValidateSession(next http.Handler) http.Handler {
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

		fmt.Println("Middleware called", cookie.Value)
		next.ServeHTTP(w, r)
	})
}

func ProduceToNSQ(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		html := `
		<h1>Protected Success</h1>
		<p>This page can only be reached when a valid crsf-token is set.</p>
		`

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Add("Random", "text/hmtl; charset=utf-8")

		w.WriteHeader(http.StatusOK)

		_, err := w.Write([]byte(html))

		if err != nil {
			SendError(w, r)
			return
		}

		return
	}
}

func FrontPageHTML(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {

		html := `
			<h1>Hello World</h1>

			<a href="/login">
				<button>Login</button>
			</a>

			<a href="/create">
				<button>CreateUser</button>
			</a>

			<a href="/protected">
				<button>Visit Protected Page</button>
			</a>


				
			<form action="/form" method="post">
				<label for="fname">First name:</label><br>
				<input type="text" id="fname" name="fname"><br>
				<label for="lname">Last name:</label><br>
				<input type="text" id="lname" name="lname">
				<input type="submit" value="Submit">
		  	</form> 
		  
		  `

		// The order in which to call these 3 is:
		//		1. Set Header
		//		2. WriteHeader
		//		3. Write
		//
		// All other cases do not work correctly!
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Add("Random", "text/hmtl; charset=utf-8")

		w.WriteHeader(http.StatusOK)

		_, err := w.Write([]byte(html))

		if err != nil {
			SendError(w, r)
			return
		}

		return
	}
	w.WriteHeader(http.StatusBadRequest)
}

func ProcessForm(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {

		err := r.ParseForm()
		if err != nil {
			SendError(w, r)
			return
		}

		fname := r.Form["fname"]
		lname := r.Form["lname"]

		log.Println("Received POST!", fname, lname)

		// http.Redirect(w, r, "/", http.StatusTemporaryRedirect)

		return
	}

	if r.Method == "GET" {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	}
	SendError(w, r)
}

func JsonPage(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {

		type sth1 struct { // Does not matter if private or not
			Zahl  int // Matters if privat or not.
			Text  string
			Datum time.Time
		}

		sth := sth1{
			Zahl:  1,
			Text:  "Hello World",
			Datum: time.Now(),
		}

		bytes, err := json.Marshal(sth)
		if err != nil {
			SendError(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, err = w.Write(bytes)
		if err != nil {
			SendError(w, r)
			return
		}
		return
	}

	w.WriteHeader(http.StatusBadRequest)
}

func Postgres() {

	url := os.Getenv("DATABASE_URL")
	fmt.Println(url)

	conn, err := pgx.Connect(context.Background(), url)
	// conn, err := pgx.Connect(context.Background(), "postgresql://postgres:postgres@localhost:5432")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())

	var name string
	var weight int64
	err = conn.QueryRow(context.Background(), "select name, weight from widgets where id=$1", 42).Scan(&name, &weight)
	if err != nil {
		fmt.Fprintf(os.Stderr, "QueryRow failed: %v\n", err)
		// os.Exit(1)
	}

	fmt.Println(name, weight)
}

func main() {

	// fmt.Println("V9:")
	// RedisUsingV9()

	err := SetupServer()
	if err != nil {
		log.Fatalln(err)
	}
	defer func() {
		server.pg.Close(context.Background())
		server.redis.Close()
	}()

	// Postgres()

	// Machtes Everything not matched somewhere else
	http.HandleFunc("/", FrontPageHTML)
	// Matches /JSON/* and redirects /JSON to /JSON/
	http.HandleFunc("/JSON/", JsonPage)
	// Matches only exaclty /Error
	http.HandleFunc("/Error", SendError)
	http.HandleFunc("/form", ProcessForm)

	http.HandleFunc("/create", CreateUser)
	http.HandleFunc("/login", LoginUser)

	http.Handle("/protected", ValidateSession(http.HandlerFunc(ProduceToNSQ)))

	log.Fatal(http.ListenAndServe(":8080", nil))

}
