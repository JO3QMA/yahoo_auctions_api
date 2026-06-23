package usecase

import (
	"context"
	"errors"
	"time"

	"jo3qma.com/yahoo_auctions/internal/domain/model"
	"jo3qma.com/yahoo_auctions/internal/domain/repository"
	"jo3qma.com/yahoo_auctions/internal/infrastructure/yahoo"
)

const (
	// DefaultLookbackDays は Lookback period の既定日数です。
	DefaultLookbackDays = 90
	// MaxComparableSearchPages は Comparable search depth の上限です。
	MaxComparableSearchPages = 3
)

// ComparableUsecase は Comparable 検索のビジネスロジックを担当します。
type ComparableUsecase struct {
	repo repository.ComparableRepository
	now  func() time.Time
}

// NewComparableUsecase は新しい ComparableUsecase を作成します。
func NewComparableUsecase(repo repository.ComparableRepository) *ComparableUsecase {
	return &ComparableUsecase{
		repo: repo,
		now:  time.Now,
	}
}

// SearchComparables は Sold search から Lookback period 内の Comparable を取得します。
func (u *ComparableUsecase) SearchComparables(
	ctx context.Context,
	categoryID string,
	keyword string,
	lookbackDays *int32,
) (*model.ComparableSearchResult, error) {
	days := int32(DefaultLookbackDays)
	if lookbackDays != nil && *lookbackDays > 0 {
		days = *lookbackDays
	}

	cutoff := u.now().AddDate(0, 0, -int(days))
	var comparables []*model.Comparable

	for page := int64(0); page < MaxComparableSearchPages; page++ {
		pageResult, err := u.repo.SearchSold(ctx, categoryID, keyword, page)
		if err != nil {
			var ue *yahoo.UpstreamError
			if errors.As(err, &ue) && ue.StatusCode == 404 {
				if page == 0 {
					return emptyComparableSearchResult(), nil
				}
				break
			}
			return nil, err
		}

		if len(pageResult.Items) == 0 {
			break
		}

		stopPaging := false
		for _, item := range pageResult.Items {
			if item.EndedAt.Before(cutoff) {
				stopPaging = true
				continue
			}
			comparables = append(comparables, item)
		}

		if stopPaging || !pageResult.HasNext {
			break
		}
	}

	return &model.ComparableSearchResult{
		Comparables: comparables,
		Count:       int64(len(comparables)),
	}, nil
}

func emptyComparableSearchResult() *model.ComparableSearchResult {
	return &model.ComparableSearchResult{
		Comparables: []*model.Comparable{},
		Count:       0,
	}
}
