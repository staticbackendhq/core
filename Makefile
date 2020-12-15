build:
	go build

start: build
	source ./.setenv.sh && ./staticbackend -host localhost

deploy:
	CGO_ENABLED=0 go build
	scp staticbackend sb-poc:/home/dstpierre/sb

test:
	@JWT_SECRET=okdevmode go test