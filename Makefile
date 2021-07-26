include .env
export $(shell sed 's/=.*//' .env)

build:
	@cd cmd && rm -rf staticbackend && go build -o staticbackend

start: build
	@./staticbackend -host localhost

deploy:
	CGO_ENABLED=0 go build
	scp staticbackend sb-poc:/home/dstpierre/sb

test:
	@JWT_SECRET=okdevmode go test --race --cover