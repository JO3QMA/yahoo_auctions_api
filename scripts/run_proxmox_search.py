#!/usr/bin/env python3
"""
プロンプト（Proxmox用ストレージサーバー）に基づき SearchAuctions を実行し、
結果を集約して必須条件・LLM依頼用に整理する。
"""
import json
import urllib.request
import urllib.error
from collections import defaultdict

BASE = "http://localhost:8080/yahoo_auction.v1.YahooAuctionService"

# 1. 検索ターゲット（本体）
QUERIES_BODY = [
    "Fujitsu TX1330 M3",
    "Fujitsu TX1330 M4",
    "Dell PowerEdge T330",
    "Dell PowerEdge T340",
    "HPE ML30 Gen9",
    "HPE ML30 Gen10",
    "NEC T110h",
    "NEC T110i",
    "NEC T110j",
    "HPE MicroServer Gen10 Plus",
]

# 2. 同時検索用パーツ
QUERIES_PARTS = [
    "Mellanox ConnectX-3 MCX311A",
    "Intel X520-DA2",
    "Intel X540-T2",
    "LSI 9211-8i IT mode",
    "LSI 9300-8i IT mode",
    "Dell H330 IT mode",
]


def search(query: str, page: int = 0) -> dict:
    url = f"{BASE}/SearchAuctions"
    data = json.dumps({"query": query, "page": page}).encode("utf-8")
    req = urllib.request.Request(
        url, data=data, method="POST", headers={"Content-Type": "application/json", "Accept": "application/json"}
    )
    with urllib.request.urlopen(req, timeout=30) as r:
        return json.loads(r.read().decode())


def get_auction(auction_id: str) -> dict | None:
    url = f"{BASE}/GetAuction"
    data = json.dumps({"auctionId": auction_id}).encode("utf-8")
    req = urllib.request.Request(
        url, data=data, method="POST", headers={"Content-Type": "application/json", "Accept": "application/json"}
    )
    try:
        with urllib.request.urlopen(req, timeout=15) as r:
            return json.loads(r.read().decode())
    except urllib.error.HTTPError as e:
        print(f"  [GetAuction error {e.code}] {auction_id}")
        return None


def main():
    seen = {}
    by_query = defaultdict(list)

    print("=== 本体（タワー型・小型サーバー）検索 ===\n")
    for q in QUERIES_BODY:
        try:
            r = search(q)
            items = r.get("items") or []
            total = r.get("totalCount", 0)
            print(f"[{q}] ヒット: {len(items)} 件 (totalCount: {total})")
            for it in items[:8]:
                aid = it.get("auctionId", "")
                if aid and aid not in seen:
                    seen[aid] = {**it, "query": q, "category": "body"}
                by_query[q].append(it)
            if len(items) > 8:
                print(f"  ... 他 {len(items) - 8} 件")
        except Exception as e:
            print(f"  ERROR: {e}")
        print()

    print("=== パーツ（10GbE・HBA/ITモード）検索 ===\n")
    for q in QUERIES_PARTS:
        try:
            r = search(q)
            items = r.get("items") or []
            total = r.get("totalCount", 0)
            print(f"[{q}] ヒット: {len(items)} 件 (totalCount: {total})")
            for it in items[:5]:
                aid = it.get("auctionId", "")
                if aid and aid not in seen:
                    seen[aid] = {**it, "query": q, "category": "parts"}
                by_query[q].append(it)
        except Exception as e:
            print(f"  ERROR: {e}")
        print()

    # 必須条件に沿ったフィルタ（タイトルベース）
    lff_keywords = ("3.5", "LFF", "3.5インチ", "4LFF", "4ベイ", "3.5inch")
    sff_exclude = ("2.5インチ", "SFF", "2.5inch", "2.5 ")
    candidates = []
    for aid, it in seen.items():
        if it.get("category") != "body":
            continue
        title = (it.get("title") or "").upper()
        title_ja = it.get("title") or ""
        if any(x in title_ja for x in sff_exclude) and not any(x in title_ja for x in lff_keywords):
            continue
        if any(x in title_ja for x in lff_keywords) or "4LFF" in title_ja:
            candidates.append(it)

    print("=== 必須条件（3.5インチ/LFF/4ベイ）に合致しそうな本体（タイトルベース） ===\n")
    for i, it in enumerate(candidates[:25], 1):
        print(f"{i}. [{it.get('auctionId')}] {it.get('currentPrice')}円")
        print(f"   {it.get('title', '')[:100]}...")
        print()

    # 上位数件の詳細を GetAuction で取得（LLM依頼用）
    print("=== 代表候補の詳細取得（GetAuction）=== \n")
    detail_ids = list({c.get("auctionId") for c in candidates[:5] if c.get("auctionId")})
    details = []
    for aid in detail_ids:
        d = get_auction(aid)
        if d:
            details.append(d)
            print(f"--- {aid} ---")
            print(f"Title: {(d.get('title') or '')[:80]}")
            print(f"Price: {d.get('currentPrice')} | Description length: {len(d.get('description') or '')}")
            print()

    # 集約結果をJSONで保存（LLM解析用）
    out = {
        "summary": {
            "body_queries": QUERIES_BODY,
            "parts_queries": QUERIES_PARTS,
            "total_unique_body_items": len([x for x in seen.values() if x.get("category") == "body"]),
            "lff_candidates_count": len(candidates),
        },
        "lff_candidates": candidates[:30],
        "details": details,
    }
    out_path = "/home/jo3qma/works/yahoo_auctions/scripts/proxmox_search_result.json"
    with open(out_path, "w", encoding="utf-8") as f:
        json.dump(out, f, ensure_ascii=False, indent=2)
    print(f"\n結果を保存しました: {out_path}")


if __name__ == "__main__":
    main()
