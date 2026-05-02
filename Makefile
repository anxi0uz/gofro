VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X github.com/anxi0uz/gofro/cmd.version=$(VERSION)"

.PHONY: build install

build:
	go build $(LDFLAGS) -o gofro .

install:
	go install $(LDFLAGS) .
