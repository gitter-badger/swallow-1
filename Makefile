GO=go

SRC = $(shell find . -type f -name '*.go' -not -path "./vendor/*")

VERSION := 1.0.0
BUILD := `git rev-parse --short HEAD`
TARGETS := task_manager task_slave
project := github.com/luocheng812/swallow

all: check build

build: $(TARGETS) $(TEST_TARGETS)

$(TARGETS): $(SRC)
	$(GO) build $(project)/cmd/$@

$(TEST_TARGETS): $(SRC)
	$(GO) build $(project)/test/$@

image: $(TARGETS)
	docker build -t manager:$(VERSION)-$(BUILD) .

.PHONY: clean all build check image

lint:
	@gometalinter --config=.gometalint ./...

packages = $(shell go list ./...|grep -v /vendor/)
test: check
	$(GO) test -v ${packages}

cov: check
	gocov test -v $(packages) | gocov-html > coverage.html

check:
	@$(GO) tool vet ${SRC}

clean:
	rm -f $(TARGETS) $(TEST_TARGETS)
