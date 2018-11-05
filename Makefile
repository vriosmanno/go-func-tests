SHELL := /bin/bash

clean: ## Remove previous build
	@rm -f bin/*

dep: ## Get the dependencies
	@go get -u -v github.com/golang/dep/cmd/dep
	@dep ensure -vendor-only

runserver:
	@dev/run_server.sh || true