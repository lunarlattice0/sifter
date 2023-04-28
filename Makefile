GO = go
GO_LDFLAGS = -s -w

all: 	clean sifter

sifter: 
	$(GO) build $(GOFLAGS) -ldflags "$(GO_LDFLAGS)"

clean: 
	rm -f sifter

.PHONY: all clean
