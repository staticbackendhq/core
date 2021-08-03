FROM golang:1.16

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
COPY .env .env

RUN go mod download

COPY . .

RUN cd cmd && go build -o ../server

CMD [ "/app/server" ]