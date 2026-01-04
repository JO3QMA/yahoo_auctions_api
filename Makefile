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
		echo "❌ golangci-lint not found."; \
		echo "   Please install it to run the same checks as CI:"; \
		echo "   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		echo "   Or using brew (macOS): brew install golangci-lint"; \
		exit 1; \
	fi

# Formatter実行
fmt:
	@echo "🎨 Running formatter..."
	$(GO) fmt ./...
	@if command -v goimports > /dev/null; then \
		goimports -w -local jo3qma.com/yahoo_auctions ./...; \
	else \
		echo "⚠️  goimports not found. Install with: go install golang.org/x/tools/cmd/goimports@latest"; \
	fi

# ヘルプ表示
help:
	@echo "Available targets:"
	@echo "  make run   - サーバーを実行します"
	@echo "  make test  - テストを実行します"
	@echo "  make lint  - Linterを実行します"
	@echo "  make fmt   - Formatterを実行します"
	@echo "  make help  - このヘルプを表示します"

