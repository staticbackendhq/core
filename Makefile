include .env
export $(shell sed 's/=.*//' .env)

build:
	@cd cmd && rm -rf staticbackend && go build -o staticbackend

start: build
	@./cmd/staticbackend

deploy: build
	scp cmd/staticbackend sb-poc:/home/dstpierre/sb
	scp -qr ./templates/* sb-poc:/home/dstpierre/templates/

alltest:
	@JWT_SECRET=okdevmode go test --race --cover ./...

thistest:
	@JWT_SECRET=okdevmode go test -run "$2" --race --cover

test-core:
	@go test --race --cover

test-pg:
	@cd database/postgresql && go test --race --cover

test-mdb:
	@cd database/mongo && go test --race --cover 

test-intl:
	@JWT_SECRET=okdevmode go test --race --cover ./internal

test-extra:
	@JWT_SECRET=okdevmode go test --race --cover ./extra

docker: build
	docker build . -t staticbackend:latest

pkg: build
	@rm -rf dist/*
	@echo "building linux binaries"
	@cd cmd && CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -o ../dist/binary-for-linux-64-bit
	@cd cmd && CGO_ENABLED=0 GOARCH=386 GOOS=linux go build -o ../dist/binary-for-linux-32-bit
	@echo "building mac binaries"
	@cd cmd && CGO_ENABLED=0 GOARCH=amd64 GOOS=darwin go build -o ../dist/binary-for-mac-64-bit
	@echo "building windows binaries"
	@cd cmd && CGO_ENABLED=0 GOARCH=amd64 GOOS=windows go build -o ../dist/binary-for-windows-64-bit.exe
	@cd cmd && CGO_ENABLED=0 GOARCH=386 GOOS=windows go build -o ../dist/binary-for-windows-32-bit.exe
	@echo "compressing binaries"
	@gzip dist/*
