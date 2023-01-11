package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-redis/redis/v9"
	"github.com/jackc/pgx/v5"
	"github.com/nsqio/go-nsq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

// TODO:
// - Deal with disconnects/reconnects with Postgres
// - Deal with disconnects/reconnects with Redis
// - Deal with disconnects/reconnects with NSQ
// - Better logging when an error happens
// - Multiple loggings create multiple session cookies

var CacheURL = os.Getenv("CACHE_URL")
var PgURL = os.Getenv("DATABASE_URL")
var NSQD = os.Getenv("NSQ_DEMON")

type Server struct {
	redis *redis.Client
	pg    *pgx.Conn
	nsq   *nsq.Producer
	mux   *chi.Mux
}

func (server *Server) SendError(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("sth went wrong"))
}

func (server *Server) SendErrorMessage(w http.ResponseWriter, r *http.Request, code int, message string) {
	w.WriteHeader(code)
	w.Write([]byte(message))
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

func AttachAllPaths(server *Server) {

	promiddleWare := NewPrometheusMiddleware("backend", []float64{5000}...) // TODO more intervals

	server.mux.Use(promiddleWare)
	// server.mux.Use(server.OtelMiddleware)

	// Machtes Everything not matched somewhere else
	server.mux.Get("/", server.FrontPageHTML)
	server.mux.Post("/form", server.ProcessForm)

	server.mux.NotFound(server.SendError)

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

	server.mux.Handle("/metrics", promhttp.Handler())
}

// Tracing stuff stolen from
// https://dev.to/aurelievache/learning-go-by-examples-part-10-instrument-your-go-app-with-opentelemetry-and-send-traces-to-jaeger-distributed-tracing-1p4a

func main() {

	server, err := SetupServer()
	if err != nil {
		log.Fatalln(err)
	}

	AttachAllPaths(server)

	// Tracer
	tp, err := tracerProvider("http://localhost:14268/api/traces")
	if err != nil {
		log.Fatal(err)
	}

	// Register our TracerProvider as the global so any imported
	// instrumentation in the future will default to using it.
	otel.SetTracerProvider(tp)

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

		tp.Shutdown(serverCtx) //TODO move to server

		err := server.Shutdown(shutdownCtx)
		if err != nil {
			log.Fatal(err)
		}
	}()

	tr := tp.Tracer("component-main")

	ctx, span := tr.Start(context.Background(), "hello")
	defer span.End()

	// HTTP Handlers
	helloHandler := func(w http.ResponseWriter, r *http.Request) {
		// Use the global TracerProvider
		tr := otel.Tracer("hello-handler")
		_, span := tr.Start(ctx, "hello")
		span.SetAttributes(attribute.Key("mykey").String("value"))
		defer span.End()

		yourName := os.Getenv("MY_NAME")
		fmt.Fprintf(w, "Hello %q!", yourName)
	}

	otelHandler := otelhttp.NewHandler(http.HandlerFunc(helloHandler), "Hello")

	server.mux.Handle("/trace", otelHandler)

	log.Fatal(http.ListenAndServe(":8080", server.mux))
}
