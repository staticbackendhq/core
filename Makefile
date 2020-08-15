s=sk_test_51DuJheLi4uPpotEYXX2iD3tbh1HKIz4x0nNcgQnNAkyKM9KwjOjt61AasorXmxNfaQyDMnW8f3BlZVyAgzlMqZP000nwgfZorR
t=okdevmode

build:
	go build

start: build
	STRIPE_KEY=$(s) JWT_SECRET=$(t) ./staticbackend -host localhost