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
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/nsqio/go-nsq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
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
var NATS_URL = os.Getenv("NATS_URL")
var GRPC_URL = os.Getenv("GRPC_URL")

func ValidateEnvVariables() {
	if CacheURL == "" {
		log.Fatal("CACHE_URL not set!")
	}
	if PgURL == "" {
		log.Fatal("DATABASE_URL not set!")
	}
	if NSQD == "" {
		log.Fatal("NSQ_DEMON not set!")
	}
	if Jaeger == "" {
		log.Fatal("JAEGER_URL not set!")
	}
	if TracingApp == "" {
		log.Fatal("TRACING_URL not set!")
	}
	if NATS_URL == "" {
		log.Fatal("NATS_URL not set!")
	}
	if GRPC_URL == "" {
		log.Fatal("GRPC_URL not set!")
	}
}

type Server struct {
	redis *redis.Client
	pg    *pgxpool.Pool
	nsq   *nsq.Producer
	mux   *chi.Mux
	tp    *trace.TracerProvider
	nats  *nats.Conn
	grpc  *grpc.ClientConn
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
	// The order in which to call these 3 is:
	//		1. Set Header
	//		2. WriteHeader
	//		3. Write
	//
	// All other cases do not work correctly!
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Add("Random", "text/hmtl; charset=utf-8")

	w.WriteHeader(http.StatusOK)

	html, err := os.ReadFile("./static/frontpage.html")
	if err != nil {
		server.SendError(w, r)
		return
	}
	_, err = w.Write(html)
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

	server.pg.Close()

	err := server.redis.Close()
	if err != nil {
		return err
	}

	err = server.tp.Shutdown(context.Background()) //TODO move to server
	if err != nil {
		return err
	}

	server.nats.Close()
	server.grpc.Close()

	log.Println("Graceful shutdown successful!")
	os.Exit(1)
	return nil
}

func AttachAllPaths(server *Server) {

	promiddleWare := NewPrometheusMiddleware("backend", []float64{400}...) // TODO more intervals

	// Prometheus Metrics
	server.mux.Use(promiddleWare)
	// All Routes are addded a span to track down requests.
	// server.mux.Use(tracing)

	server.mux.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

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
	server.mux.Post("/logout", server.LogoutUserPOST)

	server.mux.Post("/nats", server.NatsPost)
	server.mux.Post("/grpc", server.CallGRPCPost)

	// Makes it far easier to protect all underlying Handlers
	protectedRouter := chi.NewRouter()
	protectedRouter.Use(server.ValidateSession)
	protectedRouter.Get("/", server.ProduceToNSQGET)
	protectedRouter.Post("/", server.ProduceToNSQPOST)
	protectedRouter.Get("/sth", server.JsonPage)

	server.mux.Mount("/protected", protectedRouter)

}

func main() {

	ValidateEnvVariables()

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
