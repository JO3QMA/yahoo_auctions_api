package model

import "time"

// Comparable は Sold search から得られる落札済み Auction の要約です。
type Comparable struct {
	AuctionID    string
	Title        string
	WinningPrice int64
	EndedAt      time.Time
}

// ComparableSearchResult は Comparable 検索の結果です。
type ComparableSearchResult struct {
	Comparables []*Comparable
	Count       int64
}

// ComparablePage は Sold search の1ページ分の結果です。
type ComparablePage struct {
	Items   []*Comparable
	HasNext bool
}
