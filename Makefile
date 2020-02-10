is_windows := $(filter windows,$(GOOS))
SOURCES = $(patsubst ./cmd/%/main.go,%,$(shell find ./cmd -maxdepth 2 -name 'main.go'))
TARGETS = $(patsubst %,bin/%$(if $(is_windows),.exe,),$(SOURCES))
BUILD_FLAGS ?= -a -tags netgo -ldflags "-s -w"

.PHONY: all
all: build

.PHONY: build
build: $(TARGETS)

bin/%:
	CGO_ENABLED=0 go build $(BUILD_FLAGS) -o $@ cmd/$(patsubst %.exe,%,$(notdir $*))/main.go

install: build
	cp bin/* $(GOPATH)/bin/

.PHONY: clean
clean:
	rm -rf bin/ dist/
