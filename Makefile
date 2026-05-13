.PHONY: lint install-hooks

lint:
	@golangci-lint run ./...

install-hooks:
	@git config core.hooksPath .
	@echo "hooks installed"
