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
