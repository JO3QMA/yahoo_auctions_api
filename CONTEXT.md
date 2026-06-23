# Yahoo Auctions API

ヤフオクの公開ページをスクレイピングし、Connect RPC 経由でオークション情報を提供する API のドメイン用語集。

## Language

**Auction**:
ヤフオク上の1件の出品。`AuctionID`（例: `x1234567890`）で一意に識別される。List view でも Detail view でも同一の Auction である（表現の詳細度が違うだけ）。
_Avoid_: Item, 商品, Listing, CategoryItem

**List view**:
Search または Category browse によって得られる Auction の省略表現。Current price、Bid count、Buy-it-now price、サムネイル画像を含む。
_Avoid_: 一覧, CategoryItem, summary view

**Detail view**:
`GetAuction` によって得られる Auction の完全表現。説明文、複数画像、Status、Auction terms を含む。
_Avoid_: 詳細, Item, full auction

**Active**:
入札を受け付けている Auction の状態。Detail view でのみ得られる。
_Avoid_: open, 出品中

**Finished**:
ヤフオク上で終了した Auction の状態。落札・流札の区別はしない。Detail view でのみ得られる。
_Avoid_: closed, Sold, Expired, 落札済み, 流札

**Canceled**:
出品者都合などでキャンセルされた Auction の状態。Detail view でのみ得られる。
_Avoid_: cancel, 取り下げ

**Unspecified**:
スクレイプ結果から状態を判定できなかったときの Auction の状態。Detail view でのみ得られる。
_Avoid_: unknown

**Auction terms**:
Auction のスケジュールと出品ポリシー（Starting price、開始/終了日時、早期終了、自動延長、返品可否）の一式。Detail view でのみ得られる。
_Avoid_: AuctionInformation, AuctionInfo, オークション情報

**Buy-it-now price**:
即決価格。今すぐその金額で落札できる価格（単位：円）。List view でのみ得られる。設定がない Auction では `0`。
_Avoid_: ImmediatePrice, Fixed price, 定価

**Category**:
ヤフオクの商品分類体系における1つの分類。数値の Category ID（例: `2084049588`）で識別され、Category browse の閲覧軸になる。
_Avoid_: Yahoo category, Auction category, カテゴリ一覧

**Yahoo Auctions**:
この API が公開ページから Auction データを取得する外部オークションプラットフォーム（`auctions.yahoo.co.jp`）。現時点で唯一のデータ源。
_Avoid_: Upstream, Source, 外部サイト, ヤフオク（略称）

**Search**:
キーワードに基づいて Auction を List view で一覧する取得方法。`SearchAuctions` が担う。結果は新着順。
_Avoid_: Query, Keyword search, 検索API

**Category browse**:
Category に基づいて Auction を List view で一覧する取得方法。`GetCategoryItems` が担う。結果は新着順。
_Avoid_: Category search, カテゴリ検索

**Current price**:
取得時点で Yahoo Auctions に表示されている Auction の価格（単位：円）。通常は最高入札額、入札がなければ Starting price 相当の表示。List view と Detail view の両方で得られる。Yahoo Auctions が税込価格を示す場合は税込の金額を採用する。
_Avoid_: Bid price, Display price, Price, 現在価格

**Bid count**:
Auction に対する入札の件数。List view でのみ得られる。
_Avoid_: BidCount, Bids, 入札数

**Starting price**:
入札受付開始時の価格（単位：円）。Auction terms の一部。Detail view でのみ得られる。Yahoo Auctions が税込開始価格を示す場合は税込の金額を採用する。
_Avoid_: StartPrice, Initial price, 開始価格

**Page**:
Search または Category browse の結果を取得するときのページ番号。0 始まり（先頭は `0`）。1 ページあたり最大 50 件の Auction。
_Avoid_: Offset, ページ番号（1 始まり）

**Total count**:
Search または Category browse に一致する Auction の総数（Yahoo Auctions が報告する件数）。
_Avoid_: TotalCount, 件数, ヒット数

**Sold search**:
終了分検索（closedsearch）によって得られる、落札済み Auction の List view。Search や Category browse とは別系統の取得方法。
_Avoid_: Closed search, 終了分, 落札相場検索

**Winning price**:
Sold search の List view で示される落札価格（送料別、単位：円）。入札があった Auction の Current price 相当。
_Avoid_: Sold price, Final price, 落札価格, 落札額

**Comparable**:
MarketEstimate 算出に使う Sold search 結果の1件。リクエストの Category と検索キーワードに一致する Auction。
_Avoid_: Comp, 類似商品, 相場データ

**Comparable search keyword**:
Sold search に渡すキーワード文字列。呼び出し元が Product の IdentityField から導出した value をそのまま使う（例: `GTX 1080`）。API は key を検索には使わない。
_Avoid_: Query, Search term, IdentityField value

**Comparable record**:
`SearchComparables` のレスポンスで返す1件の表現。Auction ID・タイトル・Winning price・終了日時を含む Sold search 結果の要約。
_Avoid_: Sold item, 落札レコード

**Lookback period**:
Comparable 検索で対象とする落札の遡及日数。リクエストの `lookback_days` で指定し、未指定時は 90 日。
_Avoid_: Reference period, 参照期間, 検索期間

**Comparable search depth**:
Sold search を遡って取得するページ数の上限。初版は最大 3 ページ（150 件）。Lookback period 外の結果は `ended_at` で除外する。
_Avoid_: Max results, 取得件数上限

**SearchComparables**:
Category と Comparable search keyword を指定して Sold search から Comparable を取得する RPC。`YahooAuctionService` が提供する。
_Avoid_: SearchSoldComparables, GetComparables, 相場検索API

**Empty Comparable result**:
Lookback period 内に一致する Comparable が 0 件のときの正常応答。`200 OK`・空の `comparables`・`count = 0` を返し、Bot は Gemini フォールバックへ進める。
_Avoid_: Not found, 404, エラー扱い
