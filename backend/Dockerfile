# FROM golang:1.19-alpine

FROM ubuntu:latest

# Needs to be set for go mod to function
# WORKDIR /usr/games
WORKDIR /usr/src/backend
# COPY go.mod ./

# COPY go.sum ./ 

# RUN go mod download

# COPY *.go ./

# RUN go build -o backend

COPY backend /usr/src/backend/backend

EXPOSE 8080

# ENV DATABASE_URL="postgresql://postgres:postgres@db:5432"

CMD [ "./backend" ]
