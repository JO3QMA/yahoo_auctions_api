package usecase

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"jo3qma.com/yahoo_auctions/internal/domain/model"
)

type fakeCategoryRepo struct {
	page *model.CategoryItemsPage
	err  error
}

func (f fakeCategoryRepo) FetchByCategory(ctx context.Context, categoryID string, page int64) (*model.CategoryItemsPage, error) {
	return f.page, f.err
}

func TestCategoryUsecase_GetCategoryItems_delegatesToRepo(t *testing.T) {
	t.Parallel()

	expectedPage := &model.CategoryItemsPage{
		Items: []*model.CategoryItem{
			{Title: "item1"},
		},
		TotalCount: 1,
		HasNext:    false,
	}

	repo := fakeCategoryRepo{page: expectedPage}
	uc := NewCategoryUsecase(repo)

	got, err := uc.GetCategoryItems(context.Background(), "cat1", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !reflect.DeepEqual(got, expectedPage) {
		t.Errorf("got %+v, want %+v", got, expectedPage)
	}
}

func TestCategoryUsecase_GetCategoryItems_returnsRepoError(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("repo error")
	repo := fakeCategoryRepo{err: repoErr}
	uc := NewCategoryUsecase(repo)

	_, err := uc.GetCategoryItems(context.Background(), "cat1", 1)
	if !errors.Is(err, repoErr) {
		t.Errorf("got error %v, want %v", err, repoErr)
	}
}
