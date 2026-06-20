.PHONY: fmt lint fresh install-hooks release

fmt:
	@gofmt -w .

lint:
	@golangci-lint run ./...

fresh:
	@go get -u ./...
	@go mod tidy
	@echo "verifying updated dependencies..."
	@go build ./...
	@go vet ./...
	@go test ./...
	@echo "dependencies updated to latest and verified"

install-hooks:
	@git config core.hooksPath .
	@echo "hooks installed"

release:
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "error: working tree is dirty — commit all changes before releasing"; \
		exit 1; \
	fi
	@version=$$(grep -m1 -E '^## \[[0-9]+\.[0-9]+\.[0-9]+\]' CHANGELOG.md | sed -E 's/^## \[([0-9]+\.[0-9]+\.[0-9]+)\].*/\1/') && \
	if [ -z "$$version" ]; then \
		echo "error: CHANGELOG.md has no \"## [x.y.z]\" heading — add one before releasing"; \
		exit 1; \
	fi && \
	tag="v$$version" && \
	if git rev-parse "$$tag" >/dev/null 2>&1; then \
		echo "error: tag $$tag already exists — bump the CHANGELOG.md heading to the new version first"; \
		exit 1; \
	fi && \
	git tag "$$tag" && \
	git push origin "$$tag" && \
	echo "released $$tag"
