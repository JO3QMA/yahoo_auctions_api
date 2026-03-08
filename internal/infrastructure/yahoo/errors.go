package yahoo

import "fmt"

// UpstreamError はヤフオクからのHTTPレスポンスのステータスコードを保持するエラー型です。
// ハンドラー層で errors.As により取り出し、Connect のエラーコード振り分けに利用します。
type UpstreamError struct {
	StatusCode int
}

func (e *UpstreamError) Error() string {
	return fmt.Sprintf("upstream returned status %d", e.StatusCode)
}
