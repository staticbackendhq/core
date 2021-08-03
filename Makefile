include .env
export $(shell sed 's/=.*//' .env)

build:
	@cd cmd && rm -rf staticbackend && go build -o staticbackend

start: build
	@./cmd/staticbackend

deploy:
	CGO_ENABLED=0 go build
	scp staticbackend sb-poc:/home/dstpierre/sb

test:
	@JWT_SECRET=okdevmode go test --race --cover ./...

docker:
	docker build . -t staticbackend:latest

pkg: build
	@rm -rf dist/*
	@echo "building linux binaries"
	@CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -o dist/binary-for-linux-64-bit
	@CGO_ENABLED=0 GOARCH=386 GOOS=linux go build -o dist/binary-for-linux-32-bit
	@echo "building mac binaries"
	@CGO_ENABLED=0 GOARCH=amd64 GOOS=darwin go build -o dist/binary-for-mac-64-bit
	@echo "building windows binaries"
	@CGO_ENABLED=0 GOARCH=amd64 GOOS=windows go build -o dist/binary-for-windows-64-bit.exe
	@CGO_ENABLED=0 GOARCH=386 GOOS=windows go build -o dist/binary-for-windows-32-bit.exe
	@echo "compressing binaries"
	@gzip dist/*
