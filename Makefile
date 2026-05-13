.PHONY: fmt lint install-hooks release

fmt:
	@gofmt -w .

lint:
	@golangci-lint run ./...

install-hooks:
	@git config core.hooksPath .
	@echo "hooks installed"

release:
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "error: working tree is dirty — commit all changes before releasing"; \
		exit 1; \
	fi
	@version=$$(cat VERSION) && \
	tag="v$$version" && \
	git tag "$$tag" && \
	git push origin "$$tag" && \
	echo "released $$tag"
