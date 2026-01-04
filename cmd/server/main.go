package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jo3qma/protobuf/gen/go/yahoo_auction/v1/yahoo_auctionv1connect"
	"jo3qma.com/yahoo_auctions/internal/handler"
	"jo3qma.com/yahoo_auctions/internal/infrastructure/yahoo"
	"jo3qma.com/yahoo_auctions/internal/usecase"
)

func main() {
	// 依存関係の組み立て（依存性注入）
	// DBの代わりにScraperを注入することで、腐敗防止層のパターンを実現
	auctionScraper := yahoo.NewYahooScraper()          // repository.ItemRepository
	categoryScraper := yahoo.NewYahooCategoryScraper() // repository.CategoryItemRepository

	uc := usecase.NewAuctionUsecase(auctionScraper)
	catUC := usecase.NewCategoryUsecase(categoryScraper)

	h := handler.NewAuctionHandler(uc, catUC)

	// Connectハンドラーの登録
	mux := http.NewServeMux()
	path, handler := yahoo_auctionv1connect.NewYahooAuctionServiceHandler(h)
	mux.Handle(path, handler)

	// HTTPサーバーの設定
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := fmt.Sprintf(":%s", port)

	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// グレースフルシャットダウンの設定
	go func() {
		log.Printf("🚀 Server starting on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Server failed to start: %v", err)
		}
	}()

	// シグナル待機（Ctrl+Cなど）
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down server...")

	// グレースフルシャットダウン
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("❌ Server forced to shutdown: %v", err)
	}

	log.Println("✅ Server exited")
}
