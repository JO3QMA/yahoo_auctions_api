package usecase

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"jo3qma.com/yahoo_auctions/internal/domain/model"
)

type fakeSearchRepo struct {
	page *model.CategoryItemsPage
	err  error
}

func (f fakeSearchRepo) Search(ctx context.Context, query string, page int64) (*model.CategoryItemsPage, error) {
	return f.page, f.err
}

func TestSearchUsecase_Search_delegatesToRepo(t *testing.T) {
	t.Parallel()

	expectedPage := &model.CategoryItemsPage{
		Items: []*model.CategoryItem{
			{Title: "search item1", AuctionID: "s123"},
		},
		TotalCount: 1,
		HasNext:    false,
	}

	repo := fakeSearchRepo{page: expectedPage}
	uc := NewSearchUsecase(repo)

	got, err := uc.Search(context.Background(), "キーワード", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !reflect.DeepEqual(got, expectedPage) {
		t.Errorf("got %+v, want %+v", got, expectedPage)
	}
}

func TestSearchUsecase_Search_returnsRepoError(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("repo error")
	repo := fakeSearchRepo{err: repoErr}
	uc := NewSearchUsecase(repo)

	_, err := uc.Search(context.Background(), "query", 1)
	if !errors.Is(err, repoErr) {
		t.Errorf("got error %v, want %v", err, repoErr)
	}
}
