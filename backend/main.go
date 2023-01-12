package main

import (
	"context"
	"encoding/json"
	"log"
	"math/rand"
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
	"go.opentelemetry.io/otel/sdk/trace"
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
var Jaeger = os.Getenv("JAEGER_URL") //Port is 14268
var TracingApp = os.Getenv("TRACING_URL")

type Server struct {
	redis *redis.Client
	pg    *pgx.Conn
	nsq   *nsq.Producer
	mux   *chi.Mux
	tp    *trace.TracerProvider
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

			<a href="/metrics">
				<button>Prometheus metrics</button>
			</a>

			<a href="/JSON">
				<button>Sample JSON Page</button>
			</a>

			<a href="/trace">
				<button>Create a Trace</button>
			</a>
			<br><br>


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

	err = server.tp.Shutdown(context.Background()) //TODO move to server
	if err != nil {
		return err
	}

	log.Println("Graceful shutdown successful!")
	os.Exit(1)
	return nil
}

func AttachAllPaths(server *Server) {

	promiddleWare := NewPrometheusMiddleware("backend", []float64{400}...) // TODO more intervals

	// Prometheus Metrics
	server.mux.Use(promiddleWare)
	// All Routes are addded a span to track down requests.
	server.mux.Use(tracing)

	// Machtes Everything not matched somewhere else
	server.mux.Get("/", server.FrontPageHTML)
	server.mux.Post("/form", server.ProcessForm)

	// This is the default 404 Page
	server.mux.NotFound(server.SendError)

	server.mux.Get("/JSON", server.JsonPage)

	server.mux.Handle("/metrics", promhttp.Handler())
	server.mux.Get("/trace", server.SpecialTracing)

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

}

func main() {

	server, err := SetupServer()
	if err != nil {
		log.Fatalln(err)
	}

	AttachAllPaths(server)

	rand.Seed(time.Now().Unix())

	// Listen for syscall signals for process to interrupt/quit
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		s := <-sig
		log.Println("received shutdown signal: ", s)

		// Shutdown signal with grace period of 30 seconds
		shutdownCtx, cancelFunc := context.WithTimeout(context.Background(), 2*time.Second)
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
