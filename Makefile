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
	@cd plugins/topdf && CGO_ENABLE=0 go build -buildmode=plugin -o ../topdf.so

start: build
	@./cmd/staticbackend

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

test-dbs: test-pg test-mdb test-mem test-sqlite
	@echo ""

test-backend:
	@go test --cover ./backend/...

test-cache:
	@go test --cover ./cache/...

test-storage:
	@go test --cover ./storage/...

test-email:
	@go test --cover ./email/...

test-intl:
	@go test --cover ./internal

test-extra:
	@go test --cover ./extra

test-search:
	@cd search && rm -rf testdata && go test --race --cover

test-components: test-backend test-cache test-storage test-intl test-extra test-search
	@echo ""


stripe-dev:
	stripe listen --forward-to http://localhost:8099/stripe

lint:
	@golangci-lint run --timeout=10m

docker: build
	docker build . -t staticbackend:latest

pkg: build
	@rm -rf dist/*
	@echo "building linux binaries"
	@cd cmd && CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -o ../dist/binary-for-linux-64-bit
	@cd cmd && CGO_ENABLED=0 GOARCH=386 GOOS=linux go build -o ../dist/binary-for-linux-32-bit
	@echo "building mac binaries"
	@cd cmd && CGO_ENABLED=0 GOARCH=amd64 GOOS=darwin go build -o ../dist/binary-for-intel-mac-64-bit
	@cd cmd && CGO_ENABLED=0 GOARCH=arm64 GOOS=darwin go build -o ../dist/binary-for-arm-mac-64-bit
	@echo "building windows binaries"
	@cd cmd && CGO_ENABLED=0 GOARCH=amd64 GOOS=windows go build -o ../dist/binary-for-windows-64-bit.exe
	@echo copying plugins
	@cp plugins/*.so dist/
	@echo "compressing binaries"
	@gzip dist/*
