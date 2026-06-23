# SearchComparables は Sold search（closedsearch）をデータ源にする

`SearchComparables` は Active 向けの Search / Category browse ではなく、ヤフオク終了分検索（closedsearch）をスクレイプする。MarketEstimate 用の Winning price を得るには落札済み Auction が必要であり、開催中一覧からは求められないためである。

## Considered Options

- **Search / Category browse（開催中）** — 相場には使えない
- **Sold search 全件取得** — 件数が多いと遅延・負荷が大きい
- **Sold search をページ上限付きで取得（採用）** — 初版は最大 3 ページ（150 件）

## その他のトレードオフ

- **Lookback period 既定 90 日**: Bot 要件。Yahoo の終了分検索は最大 180 日分を表示するが、API は `ended_at` で 90 日（`lookback_days` 未指定時）に絞る。
- **パース方式**: closedsearch は React 配信のため、HTML の `__NEXT_DATA__` JSON から `search.items.listing.items` を読む。開催中一覧と同じ `Product__*` DOM パースは使わない。
- **Identity field key**: リクエストに含めるが検索には使わない。Comparable search keyword は value のみ。
