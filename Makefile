PACKAGE  = $(shell basename "$(PWD)")
DATE    ?= $(shell date +%FT%T%z)
VERSION ?= $(shell git describe --tags --always --dirty --match=v* 2> /dev/null || \
			cat $(CURDIR)/.version 2> /dev/null || echo v0)
GOPATH   = $(CURDIR)/.gopath
BIN      = $(GOPATH)/bin
BASE     = $(GOPATH)/src/$(PACKAGE)
PKGS     = $(or $(PKG),$(shell cd $(BASE) && env GOPATH=$(GOPATH) $(GO) list ./... | grep -v "^$(PACKAGE)/vendor/"))

export GOPATH

GO      = go
GODOC   = godoc
TIMEOUT = 15
V = 0
Q = $(if $(filter 1,$V),,@)
M = $(shell printf "\033[34;1m▶\033[0m")

.PHONY: all
all: vendor replace-btcd | $(BASE) ; $(info $(M) xgo building executable…) @ ## Build program binary
	$Q cd $(BASE) && xgo --targets=linux/amd64 --dest=$(BASE)/bin $(BASE)/cmd/wallet_tools

$(BASE): ; $(info $(M) setting GOPATH…)
	@mkdir -p $(dir $@)
	@ln -sf $(CURDIR) $@

# Tools
$(BIN):
	@mkdir -p $@
$(BIN)/%: | $(BIN) ; $(info $(M) building $(REPOSITORY)…)
	$Q tmp=$$(mktemp -d); \
	   env GO111MODULE=off GOCACHE=off GOPATH=$$tmp GOBIN=$(BIN) $(GO) get $(REPOSITORY) \
		|| ret=$$?; \
	   rm -rf $$tmp ; exit $$ret

GOXGO = $(BIN)/xgo
$(BIN)/xgo: REPOSITORY = github.com/karalabe/xgo

GODEP = $(BIN)/dep
$(BIN)/dep: REPOSITORY = github.com/golang/dep/cmd/dep

# Dependency management
vendor: Gopkg.toml Gopkg.lock | $(BASE) $(GODEP) ; $(info $(M) retrieving dependencies…)
	$Q cd $(BASE) && $(GODEP) ensure -v
	@touch $@

# replace btcsuite/btcd with wenweih/btcd_m_backup
replace-btcd: ; $(info $(M) replace btcsuite/btcd with wenweih/btcd_m_backup…)
	@rm -rf $(BASE)/vendor/github.com/btcsuite/btcd/
	@git clone https://github.com/wenweih/btcd_m_backup.git $(BASE)/vendor/github.com/btcsuite/btcd/

# Misc

.PHONY: clean
clean: ; $(info $(M) cleaning…)	@ ## Cleanup everything
	@rm -rf $(GOPATH)
	@rm -rf bin

.PHONY: help
help:
	@grep -E '^[ a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

.PHONY: version
version:
	@echo $(VERSION)
