.PHONY: build run remove

imagename=simpleservergo
imageversion=v1

# @ removes the line where it prints the command
default:	
	@echo "You can run 'build', 'run' and 'clean'" 

# Creating the image:
build:
	docker compose build

# Starting a container which will be deleted after exit:
run:
	# docker run --rm -p 8080:8080 ${imagename}:${imageversion}
	docker compose --rm up
 
# docker image rm ${imagename}:${imageversion}
clean:
	
	docker compose down
	docker rmi $(docker images | grep 'codingchallengehans')



