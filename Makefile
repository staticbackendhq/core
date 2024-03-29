include .env
export $(shell sed 's/=.*//' .env)

cleanup:
	@rm -rf dev.db && rm -rf backend/dev.db
	@rm -rf *.fts && rm -rf backend/*.fts

build:
	@cd cmd && rm -rf staticbackend && go build \
	-ldflags "-X github.com/staticbackendhq/core/config.BuildTime=$(shell date +'%Y-%m-%d.%H:%M:%S') \
	-X github.com/staticbackendhq/core/config.CommitHash=$(shell git log --pretty=format:'%h' -n 1) \
	-X github.com/staticbackendhq/core/config.Version=$(shell git describe --tags)" \
	-o staticbackend

start: build
	@./cmd/staticbackend

deploy: build
	scp cmd/staticbackend sb-poc:/home/dstpierre/sb
	scp -qr ./templates/* sb-poc:/home/dstpierre/templates/
	scp -qr ./sql/* sb-poc:/home/dstpierre/sql/

alltest:
	@go clean -testcache && go test --cover ./...

thistest:
	go test -run $(TESTNAME) --cover

test-core: cleanup
	@go clean -testcache && go test --cover

test-pg:
	@cd database/postgresql && go test --race --cover

test-mdb:
	@cd database/mongo && go test --race --cover 

test-mem:
	@rm -rf database/memory/mem.db
	@go test --race --cover ./database/memory

test-sqlite:
	@cd database/sqlite && go test --cover

test-intl:
	@JWT_SECRET=okdevmode go test --race --cover ./internal

test-extra:
	@JWT_SECRET=okdevmode go test --race --cover ./extra

test-search:
	@cd search && rm -rf testdata && go test --race --cover

stripe-dev:
	stripe listen -p sb --forward-to http://localhost:8099/stripe

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
	@echo "compressing binaries"
	@gzip dist/*
