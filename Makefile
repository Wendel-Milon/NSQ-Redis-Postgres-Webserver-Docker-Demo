.PHONY: build clean superclean

.EXPORT_ALL_VARIABLES:
DATABASE_URL=postgresql://postgres:postgres@localhost:5432
CACHE_URL=localhost
TRACING_URL=localhost:8001
JAEGER_URL=localhost:14268
NSQ_DEMON=localhost

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

# Starting a container which will be deleted after exit:
# run:
# 	# docker run --rm -p 8080:8080 ${imagename}:${imageversion}
# 	docker compose --rm up
 
# docker image rm ${imagename}:${imageversion}


run:
	@cd backend; go run .


	
clean:
	docker compose down
	cd nsqconsumer; rm nsqconsumer
	cd backend; rm backend
	cd tracingApp; rm tracingapp


superclean: clean
	docker rmi codingchallengehans-gobackend:latest
	docker rmi codingchallengehans-tracingapp:latest
	docker rmi codingchallengehans-nsqconsumer_links:latest
	docker rmi codingchallengehans-nsqconsumer_rechts:latest

	docker volume rm codingchallengehans_cache
	docker volume rm codingchallengehans_db




