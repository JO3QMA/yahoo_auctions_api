.PHONY: run test lint help

# デフォルトターゲット
.DEFAULT_GOAL := help

# 変数定義
GO := go
GOLANGCI_LINT := golangci-lint
SERVER_MAIN := cmd/server/main.go

# サーバー実行
run:
	@echo "🚀 Starting server..."
	$(GO) run $(SERVER_MAIN)

# テスト実行
test:
	@echo "🧪 Running tests..."
	$(GO) test -v ./...

# Linter実行
lint:
	@echo "🔍 Running linter..."
	@if command -v $(GOLANGCI_LINT) > /dev/null; then \
		$(GOLANGCI_LINT) run; \
	else \
		echo "⚠️  golangci-lint not found. Running go vet and go fmt instead..."; \
		$(GO) vet ./...; \
		$(GO) fmt ./...; \
	fi

# ヘルプ表示
help:
	@echo "Available targets:"
	@echo "  make run   - サーバーを実行します"
	@echo "  make test  - テストを実行します"
	@echo "  make lint  - Linterを実行します"
	@echo "  make help  - このヘルプを表示します"

