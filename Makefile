OUTPUT = draft

all: build

test:
	go vet && go test

build:
ifndef DRAFT_BUILD_VERSION
	$(error DRAFT_BUILD_VERSION is not set. Please export DRAFT_BUILD_VERSION as an environment variable.)
endif
	go build -ldflags "-X main.Version=$(DRAFT_BUILD_VERSION) -X main.BuildDate=`date +%Y%m%d%H%M`" -o $(OUTPUT)

clean:
	rm -f $(OUTPUT)

.PHONY: all build clean

