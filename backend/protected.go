package main

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog/log"

	"go.opentelemetry.io/otel/propagation"
)

func (server *Server) ValidateSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		cookie, err := r.Cookie("csrftoken")
		if err != nil {
			log.Info().Msgf("Middleware Validate caught csrf-token not set %v", err)
			http.Redirect(w, r, "/login", http.StatusUnauthorized)
			return
		}

		_, err = server.redis.Get(context.Background(), cookie.Value).Result()
		if err != nil {
			log.Info().Msgf("Middleware Validate caught csrf-token does not exist %v", err)
			http.Redirect(w, r, "/login", http.StatusUnauthorized)
			return
		}

		// log.Info().Msgf("Middleware called", cookie.Value)
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
		log.Warn().Err(err).Caller().Msg("")
		server.SendError(w, r)
		return
	}
}

func (server *Server) ProduceToNSQPOST(w http.ResponseWriter, r *http.Request) {

	ctx, span := server.tp.Tracer("NSQ-Producer").Start(context.Background(), "Start")
	defer span.End()

	// https://stackoverflow.com/questions/71895937/manually-extracting-opentelemetry-context-from-golang-into-a-string
	carrier := propagation.MapCarrier{}
	propagator := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
	propagator.Inject(ctx, carrier)

	// log.Info().Msgf(carrier)

	message, err := json.Marshal(carrier)
	if err != nil {
		log.Info().Err(err).Msg("")
	}

	//TODO enable selection of topic and message
	// message := "default message"

	// err := server.nsq.Publish("default", []byte(message))
	err = server.nsq.Publish("default", message)
	if err != nil {
		server.SendErrorMessage(w, r, http.StatusBadRequest, err.Error())
		log.Info().Msgf("Error when producing message %v", err)
		return
	}
	log.Info().Msg("Succesfully produced message")
}
