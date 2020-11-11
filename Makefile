build:
	go build

start: build
	source ./.setenv.sh && ./staticbackend -host localhost