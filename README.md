# NSQ-Redis-Postgres-Webserver-Docker-Demo

## Start via
1) make build
2) docker compose up

**Warning!** Backend service might crash on initial startup. \
Because PostgreSQL does some wierd shutdown/restart shannaigans. \
to solve simply restart container.

## Services

### Backend
 - Handles all the Logic.
 - Produces NSQ Messages
 - Available Routes:
   - / -> default
   - /login -> sets cookie for /protected
   - /logout
   - /create -> Create a new User
   - /protected -> Can only be accessed via crsf-token
   - /JSON -> just some example JSON
   - /form -> deals with the Form on default page

### PostgreSQL 
 - stores the user via UserID and bycrpt encrypted Password
 - uses *db.sql* file to setup new Tables
 - preservs state via volume

### Redis
 - stores a uuid4 which should represent a session cookie
 - expires after 10 Minutes
 - preservs state via volume

### NSQ 
 - single Deamon & Lookup setup
 - no data backup

### NSQ Consumer
 - Cosumes all Messages in topic "default"
 - There are 2 Channels on the "default" topic
 - On each channel there are two consumers
 - simulate work by sleeping random time
 - does nothing but print message

### TracingApp
 - simply there to test tracing via Jaeger

## Metrics 
Included in the Docker compose is a Prometheus Instance which periodically queries, most Services. \
Special case *NSQ_CONSUMER* it is configured to query all instances spawned by the *replicas* in the Docker Compose file.

## Tracing
Included in the Docker compose is a Jaeger Tracing Instance which can visualize the Traces. \
The gobackend has a middleware which creates a trace for every route, which is quite pointless. \
When querying "/tracing" or producing a NSQ message, a more usefull trace is produced. \
  
## Makefile
I changed to Dockerfile from default golang:alpine to ubuntu:latest
As a result the go project needs to be precompiled. To that via *make build*
The resulting image is **70mb** instead of **300mb**.

There is also a Dockerfile for alpine. When build this way the final image is only **20mb**.
However, the go project needs to build with *CGO_ENABLED=0 go build .*

## Github Actions
 - single actions which builds images the correct way.

## TODO
- Reconnects
- Logging
- Tests
- 