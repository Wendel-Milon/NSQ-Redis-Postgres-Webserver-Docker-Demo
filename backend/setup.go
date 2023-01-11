package main

import (
	"context"
	"fmt"
	"log"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-redis/redis/v9"
	"github.com/jackc/pgx/v5"
	"github.com/nsqio/go-nsq"
)

func SetupServer() (*Server, error) {

	/************************ REDIS ***************************/

	rdb, err := ConnectRedis()
	if err != nil {
		return nil, err
	}

	/************************** POSTGRES ***********************/

	pgconn, err := ConnectPostgre()
	if err != nil {
		return nil, err
	}

	/************************** NSQ **********************************/

	nsq, err := ConnectNSQ()
	if err != nil {
		return nil, err
	}

	/************************** Chi MUX *********************************/

	mux := CreateRouter()

	/*****************/

	return &Server{
		redis: rdb,
		pg:    pgconn,
		nsq:   nsq,
		mux:   mux,
	}, nil
}

func ConnectRedis() (*redis.Client, error) {

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

	log.Println("Successfully connected to Redis")
	return rdb, nil
}

func ConnectPostgre() (*pgx.Conn, error) {
	conn, err := pgx.Connect(context.Background(), PgURL)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %v", err)
	}

	log.Println("Successfully connected to Postgres")

	return conn, nil
}

func ConnectNSQ() (*nsq.Producer, error) {
	config := nsq.NewConfig()
	producer, err := nsq.NewProducer(fmt.Sprintf("%s:4150", NSQD), config)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to NSQ Demon %v", err)
	}

	log.Println("Successfully connected to NSQDemon")
	return producer, nil
}

func CreateRouter() *chi.Mux {
	mux := chi.NewRouter()

	// Makes it far easier to implement Middleware for all Routes.
	// A good base middleware stack
	mux.Use(middleware.RequestID) // Injects a request ID into the context of each request
	mux.Use(middleware.RealIP)    // Sets a http.Request's RemoteAddr to either X-Real-IP or X-Forwarded-For
	mux.Use(middleware.Logger)    // Logs the start and end of each request with the elapsed processing time
	mux.Use(middleware.Recoverer) // Gracefully absorb panics and prints the stack trace

	return mux
}
