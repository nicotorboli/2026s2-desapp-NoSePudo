package repository_test

import (
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/dao"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/repository"
)

func newHistogramRepo(t *testing.T) *repository.ScoreHistogramRepositoryImpl {
	t.Helper()
	return repository.NewScoreHistogramRepository(dao.NewScoreHistogramDao(startPostgres(t)))
}

func TestScoreHistogramSaveAndGet(t *testing.T) {
	repo := newHistogramRepo(t)
	ctx := t.Context()

	want := []int64{0, 3, 7, 0}
	id, err := repo.Save(ctx, model.ScoreHistogram{Position: "Midfielder", LeagueCode: "PL", Buckets: want})
	if err != nil {
		t.Fatalf("Save devolvió error: %v", err)
	}
	if id == 0 {
		t.Error("Save no devolvió identificador")
	}

	got, err := repo.Get(ctx, model.ScoreScope{Position: "Midfielder", LeagueCode: "PL"})
	if err != nil {
		t.Fatalf("Get devolvió error: %v", err)
	}
	if len(got.Buckets) != len(want) {
		t.Fatalf("buckets = %v, want %v", got.Buckets, want)
	}
	for i := range want {
		if got.Buckets[i] != want[i] {
			t.Fatalf("bucket %d = %d, want %d", i, got.Buckets[i], want[i])
		}
	}
}

func TestScoreHistogramGetReturnsLatest(t *testing.T) {
	repo := newHistogramRepo(t)
	ctx := t.Context()

	if _, err := repo.Save(ctx, model.ScoreHistogram{Position: "Goalkeeper", LeagueCode: "PL", Buckets: []int64{1, 0}}); err != nil {
		t.Fatalf("primer Save: %v", err)
	}
	if _, err := repo.Save(ctx, model.ScoreHistogram{Position: "Goalkeeper", LeagueCode: "PL", Buckets: []int64{0, 2}}); err != nil {
		t.Fatalf("segundo Save: %v", err)
	}

	got, err := repo.Get(ctx, model.ScoreScope{Position: "Goalkeeper", LeagueCode: "PL"})
	if err != nil {
		t.Fatalf("Get devolvió error: %v", err)
	}
	if got.Buckets[1] != 2 {
		t.Errorf("buckets = %v, se esperaba el más reciente [0 2]", got.Buckets)
	}
}

func TestScoreHistogramGetMissingScope(t *testing.T) {
	repo := newHistogramRepo(t)

	if _, err := repo.Get(t.Context(), model.ScoreScope{Position: "Goalkeeper", LeagueCode: "XX"}); err == nil {
		t.Fatal("Get de un scope sin histograma no devolvió error")
	}
}
