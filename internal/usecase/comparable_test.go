package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"jo3qma.com/yahoo_auctions/internal/domain/model"
	"jo3qma.com/yahoo_auctions/internal/infrastructure/yahoo"
)

type fakeComparableRepo struct {
	pages [][]*model.Comparable
	err   error
	calls int
}

func (f *fakeComparableRepo) SearchSold(ctx context.Context, categoryID, keyword string, page int64) (*model.ComparablePage, error) {
	if f.err != nil {
		return nil, f.err
	}
	if int(page) >= len(f.pages) {
		return &model.ComparablePage{}, nil
	}
	items := f.pages[page]
	f.calls++
	return &model.ComparablePage{
		Items:   items,
		HasNext: len(items) >= 50,
	}, nil
}

func TestComparableUsecase_SearchComparables_filtersLookbackAndStopsPaging(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)
	inRange := now.AddDate(0, 0, -30)
	outOfRange := now.AddDate(0, 0, -120)

	repo := &fakeComparableRepo{
		pages: [][]*model.Comparable{
			{
				{AuctionID: "a1", WinningPrice: 100, EndedAt: inRange},
				{AuctionID: "a2", WinningPrice: 200, EndedAt: outOfRange},
			},
		},
	}

	uc := NewComparableUsecase(repo)
	uc.now = func() time.Time { return now }

	result, err := uc.SearchComparables(context.Background(), "2084040405", "GTX 1080", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 1 {
		t.Fatalf("Count got %d, want 1", result.Count)
	}
	if result.Comparables[0].AuctionID != "a1" {
		t.Fatalf("AuctionID got %q, want a1", result.Comparables[0].AuctionID)
	}
	if repo.calls != 1 {
		t.Fatalf("repo calls got %d, want 1", repo.calls)
	}
}

func TestComparableUsecase_SearchComparables_emptyOn404(t *testing.T) {
	t.Parallel()

	repo := &fakeComparableRepo{err: &yahoo.UpstreamError{StatusCode: 404}}
	uc := NewComparableUsecase(repo)

	result, err := uc.SearchComparables(context.Background(), "2084040405", "missing", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 0 {
		t.Fatalf("Count got %d, want 0", result.Count)
	}
}

func TestComparableUsecase_SearchComparables_propagatesNon404Error(t *testing.T) {
	t.Parallel()

	repo := &fakeComparableRepo{err: errors.New("upstream")}
	uc := NewComparableUsecase(repo)

	_, err := uc.SearchComparables(context.Background(), "2084040405", "GTX 1080", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
