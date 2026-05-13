.PHONY: fmt lint install-hooks

fmt:
	@gofmt -w .

lint:
	@golangci-lint run ./...

install-hooks:
	@git config core.hooksPath .
	@echo "hooks installed"
