IMAGE := codingbox/sandbox:latest
MAIN := ./cmd/codingbox

.PHONY: build

build:
	docker build -t $(IMAGE) .
	GOBIN=$(HOME)/.local/bin go install $(MAIN)
