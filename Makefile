.PHONY: build clean superclean

.EXPORT_ALL_VARIABLES:
DATABASE_URL=postgresql://postgres:postgres@localhost:5432
CACHE_URL=localhost
TRACING_URL=localhost:8001
JAEGER_URL=localhost:14268
NSQ_DEMON=localhost
NATS_URL=localhost
MY_NAME=günther

imagename=simpleservergo
imageversion=v1

# @ removes the line where it prints the command
default:	
	@echo "You can run 'build', 'run' and 'clean'" 

# Creating the image:
build:
	cd nsqconsumer; go build -o nsqconsumer .
	cd backend; go build -o backend .
	cd tracingApp; go build -o tracingapp .
	cd natsconsumer;CGO_ENABLED=0 go build -o natsconsumer . 
	cd grpcconsumer;CGO_ENABLED=0 go build -o grpcconsumer . 

tidy:
	cd nsqconsumer; go mod tidy
	cd backend; go mod tidy
	cd tracingApp; go mod tidy
	cd natsconsumer; go mod tidy
	cd grpcconsumer; go mod tidy

run:
	@cd backend; go run .


	
clean:
	docker compose down
	cd nsqconsumer; rm nsqconsumer
	cd backend; rm backend
	cd tracingApp; rm tracingapp
	cd natsconsumer; rm natsconsumer
	cd grpcconsumer; rm grpcconsumer


superclean: clean
	docker rmi coding_challenge-gobackend:latest
	docker rmi coding_challenge-tracingapp:latest
	docker rmi coding_challenge-nsqconsumer_links:latest
	docker rmi coding_challenge-nsqconsumer_rechts:latest
	docker rmi coding_challenge-natsconsumer:latest

	docker volume rm coding_challenge_cache
	docker volume rm coding_challenge_db
	docker volume rm coding_challenge_prometheus_data




