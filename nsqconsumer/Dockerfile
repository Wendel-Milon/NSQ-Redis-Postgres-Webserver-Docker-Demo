FROM ubuntu:latest

# Needs to be set for go mod to function
# WORKDIR /usr/games

# COPY go.mod ./

# COPY go.sum ./ 

# RUN go mod download

# COPY *.go ./

# RUN go build -o nsqconsumer

WORKDIR /usr/src/nsqconsumer

COPY nsqconsumer /usr/src/nsqconsumer/nsqconsumer

EXPOSE 8080

# ENV DATABASE_URL="postgresql://postgres:postgres@db:5432"

CMD [ "./nsqconsumer" ]
