package model

// Item はオークション商品のドメインモデルです
// 外部サイト（ヤフオク）のHTML構造を知らない、純粋なデータ構造を定義します
type Item struct {
	AuctionID    string
	Title        string
	CurrentPrice int64  // 現在価格（単位：円）
	ShippingFee  int64  // 送料（単位：円）
	Status       Status // オークションの状態
}

// Status はオークションの状態を表します
type Status int32

const (
	StatusUnspecified Status = 0
	StatusActive      Status = 1 // 出品中（入札可能な状態）
	StatusFinished    Status = 2 // 終了済み（落札または時間切れ）
	StatusCanceled    Status = 3 // 出品者都合などでキャンセルされた状態
)
