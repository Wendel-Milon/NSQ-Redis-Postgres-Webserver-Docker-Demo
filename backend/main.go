package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-redis/redis/v9"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/nsqio/go-nsq"
	"golang.org/x/crypto/bcrypt"
)

var CacheURL = os.Getenv("CACHE_URL")
var PgURL = os.Getenv("DATABASE_URL")
var NSQD = os.Getenv("NSQ_DEMON")

type Server struct {
	redis *redis.Client
	pg    *pgx.Conn
	nsq   *nsq.Producer
	mux   *chi.Mux
}

func SetupServer() (*Server, error) {

	server := Server{}

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
		return nil, fmt.Errorf("unable to set redis value: %v", err)

	}
	err = rdb.Del(context.Background(), "TEST").Err()
	if err != nil {
		return nil, fmt.Errorf("unable to delete the redis key: %v", err)
	}
	server.redis = rdb

	log.Println("Successfully connected to Redis")

	/************************** POSTGRES ***********************/

	conn, err := pgx.Connect(context.Background(), PgURL)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %v", err)
	}
	server.pg = conn

	log.Println("Successfully connected to Postgres")

	/************************** NSQ **********************************/

	config := nsq.NewConfig()
	producer, err := nsq.NewProducer(fmt.Sprintf("%s:4150", NSQD), config)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to NSQ Demon %v", err)
	}

	server.nsq = producer

	log.Println("Successfully connected to NSQDemon")

	/************************** Chi MUX *********************************/

	server.mux = chi.NewRouter()

	// Makes it far easier to implement Middleware for all Routes.
	// A good base middleware stack
	server.mux.Use(middleware.RequestID) // Injects a request ID into the context of each request
	server.mux.Use(middleware.RealIP)    // Sets a http.Request's RemoteAddr to either X-Real-IP or X-Forwarded-For
	server.mux.Use(middleware.Logger)    // Logs the start and end of each request with the elapsed processing time
	server.mux.Use(middleware.Recoverer) // Gracefully absorb panics and prints the stack trace

	// Makes it simple to write a catch all 404 Page.
	server.mux.NotFound(server.SendError)

	return &server, nil
}

func (server *Server) SendError(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("sth went wrong"))
}

func (server *Server) SendErrorMessage(w http.ResponseWriter, r *http.Request, message string) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(message))
}

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
		server.SendError(w, r)
		return
	}
}

func (server *Server) CreateUserPOST(w http.ResponseWriter, r *http.Request) {
	// if r.Method == "POST" {

	err := r.ParseForm()
	if err != nil {
		server.SendErrorMessage(w, r, err.Error())
		return
	}

	userid, ok := r.Form["userid"]
	if !ok {
		server.SendErrorMessage(w, r, "notOK") // Maybe just index array....
		return
	}
	joinedUser := strings.Join(userid, "")

	passwd, ok := r.Form["passwd"]
	if !ok {
		server.SendErrorMessage(w, r, "notOK")
		return
	}
	joined := strings.Join(passwd, "") // Maybe just index array....

	//TODO Check for userId already exits!.

	hash, err := bcrypt.GenerateFromPassword([]byte(joined), bcrypt.DefaultCost)
	if err != nil {
		server.SendErrorMessage(w, r, err.Error())
		return
	}

	sql := `INSERT INTO users (userid, passwd) VALUES ($1, $2)`

	_, err = server.pg.Exec(context.Background(), sql, joinedUser, hash)
	if err != nil {
		server.SendErrorMessage(w, r, err.Error())
		return
	}

	log.Println("User successfully created", userid)
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
		server.SendError(w, r)
		return
	}
}

func (server *Server) LoginUserPOST(w http.ResponseWriter, r *http.Request) {

	err := r.ParseForm()
	if err != nil {
		server.SendErrorMessage(w, r, err.Error())
		return
	}

	userid, ok := r.Form["userid"]
	if !ok {
		server.SendErrorMessage(w, r, "notOK") // Maybe just index array....
		return
	}
	joinedUser := strings.Join(userid, "")

	passwd, ok := r.Form["passwd"]
	if !ok {
		server.SendErrorMessage(w, r, "notOK")
		return
	}
	joined := strings.Join(passwd, "") // Maybe just index array....

	var pwhash []byte
	err = server.pg.QueryRow(context.Background(), "select passwd FROM users where userid=$1", joinedUser).Scan(&pwhash)
	if err != nil {
		server.SendErrorMessage(w, r, err.Error())
		return
	}

	err = bcrypt.CompareHashAndPassword(pwhash, []byte(joined))
	if err != nil {
		server.SendErrorMessage(w, r, err.Error())
		return
	}

	// Create Cookie
	token, err := uuid.NewRandom()
	if err != nil {
		server.SendErrorMessage(w, r, err.Error())
		return
	}
	// Add to Redis
	err = server.redis.Set(context.Background(), token.String(), token, time.Minute*10).Err()
	if err != nil {
		server.SendErrorMessage(w, r, err.Error())
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
}

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
		server.SendErrorMessage(w, r, err.Error())
		log.Println("Error when producing message", err)
		return
	}
	fmt.Println("Succesfully produced message")
}

func (server *Server) FrontPageHTML(w http.ResponseWriter, r *http.Request) {
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
			<br>


			<p> This forms does nothing.</p>	
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
		server.SendError(w, r)
		return
	}
}

func (server *Server) ProcessForm(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		server.SendError(w, r)
		return
	}

	fname := r.Form["fname"]
	lname := r.Form["lname"]

	log.Println("Received POST!", fname, lname)
}

func (server *Server) JsonPage(w http.ResponseWriter, r *http.Request) {

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
		server.SendError(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_, err = w.Write(bytes)
	if err != nil {
		server.SendError(w, r)
		return
	}
}

func (server *Server) Shutdown(context.Context) error {

	err := server.pg.Close(context.Background())
	if err != nil {
		return err
	}
	err = server.redis.Close()
	if err != nil {
		return err
	}

	log.Println("Graceful shutdown successful!")
	os.Exit(1)
	return nil
}

func main() {

	server, err := SetupServer()
	if err != nil {
		log.Fatalln(err)
	}

	// Machtes Everything not matched somewhere else
	server.mux.Get("/", server.FrontPageHTML)
	server.mux.Post("/form", server.ProcessForm)

	// Matches /JSON/* and redirects /JSON to /JSON/
	server.mux.Get("/JSON/", server.JsonPage)

	// Matches only exaclty /Error
	server.mux.HandleFunc("/Error", server.SendError)

	// Makes the Handlers far simpler and easier to understand
	// And also far smaller.
	server.mux.Get("/create", server.CreateUserGET)
	server.mux.Post("/create", server.CreateUserPOST)

	server.mux.Get("/login", server.LoginUserGET)
	server.mux.Post("/login", server.LoginUserPOST)

	// Makes it far easier to protect all underlying Handlers
	protectedRouter := chi.NewRouter()

	protectedRouter.Use(server.ValidateSession)
	protectedRouter.Get("/", server.ProduceToNSQGET)
	protectedRouter.Post("/", server.ProduceToNSQPOST)
	protectedRouter.Get("/sth", server.JsonPage)

	server.mux.Mount("/protected", protectedRouter)

	// Server run context
	serverCtx, serverStopCtx := context.WithCancel(context.Background())

	// Listen for syscall signals for process to interrupt/quit
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		s := <-sig
		log.Println("received shutdown signal: ", s)

		// Shutdown signal with grace period of 30 seconds
		shutdownCtx, cancelFunc := context.WithTimeout(serverCtx, 2*time.Second)
		defer serverStopCtx()
		defer cancelFunc()

		go func() {
			<-shutdownCtx.Done()
			if shutdownCtx.Err() == context.DeadlineExceeded {
				log.Fatal("graceful shutdown timed out.. forcing exit.")
			}
		}()

		// Trigger graceful shutdown
		err := server.Shutdown(shutdownCtx)
		if err != nil {
			log.Fatal(err)
		}
	}()

	log.Fatal(http.ListenAndServe(":8080", server.mux))
}
