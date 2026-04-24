IMAGE := codingbox/sandbox:latest
BINARY := codingbox
MAIN := ./cmd/codingbox

.PHONY: build

build:
	docker build -t $(IMAGE) .
	go build -o $(BINARY) $(MAIN)
