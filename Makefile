GO=go
BINDIR=./bin
ARMDIR=$(BINDIR)/arm
X86DIR=$(BINDIR)/x86
BINARIES = inserter reader

all: build
build: ${BINARIES}

${BINARIES}:
	GOOS=linux GOARCH=arm $(GO) build -o $(ARMDIR)/$@ ./cmd/$@
	$(GO) build -o $(X86DIR)/$@ ./cmd/$@

run-reader: build
	$(X86DIR)/reader

run-inserter: build
	$(ARMDIR)/inserter
