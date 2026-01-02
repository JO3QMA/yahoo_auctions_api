package model

import "time"

// Item はオークション商品のドメインモデルです
// 外部サイト（ヤフオク）のHTML構造を知らない、純粋なデータ構造を定義します
type Item struct {
	AuctionID    string
	Title        string
	CurrentPrice int64               // 現在価格（単位：円）
	ShippingFee  int64               // 送料（単位：円）
	Status       Status              // オークションの状態
	Images       []string            // 商品画像のURLリスト
	AuctionInfo  *AuctionInformation // オークション情報
	Description  string              // 商品説明（HTML）
}

// AuctionInformation はオークションの詳細情報を表します
type AuctionInformation struct {
	AuctionID        string    // オークションID
	StartPrice       int64     // 開始価格（単位：円）
	StartTime        time.Time // 開始日時
	EndTime          time.Time // 終了日時
	EarlyEnd         bool      // 早期終了
	AutoExtension    bool      // 自動延長
	Returnable       bool      // 返品の可否
	ReturnableDetail string    // 返品の可否（詳細）
}

// Status はオークションの状態を表します
type Status int32

const (
	StatusUnspecified Status = 0
	StatusActive      Status = 1 // 出品中（入札可能な状態）
	StatusFinished    Status = 2 // 終了済み（落札または時間切れ）
	StatusCanceled    Status = 3 // 出品者都合などでキャンセルされた状態
)
