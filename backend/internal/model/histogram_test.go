package model

import "testing"

func TestBucketIndex(t *testing.T) {
	cases := []struct {
		name    string
		score   float64
		buckets int
		want    int
	}{
		{"cero", 0.0, 1000, 0},
		{"uno", 1.0, 1000, 999},
		{"medio", 0.5, 1000, 500},
		{"negativo", -0.1, 1000, 0},
		{"mayor que uno", 1.5, 1000, 999},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := bucketIndex(c.score, c.buckets); got != c.want {
				t.Fatalf("bucketIndex(%v, %d) = %d, want %d", c.score, c.buckets, got, c.want)
			}
		})
	}
}

func TestBuildHistogram(t *testing.T) {
	got := BuildHistogram([]float64{0.0, 0.0, 1.0}, 2)
	want := []int64{2, 1}

	if len(got) != len(want) {
		t.Fatalf("longitud = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("bucket %d = %d, want %d", i, got[i], want[i])
		}
	}
}

func TestPercentile(t *testing.T) {
	hist := []int64{2, 1} // 3 elementos: dos en bucket 0, uno en bucket 1

	cases := []struct {
		name string
		raw  float64
		want float64
	}{
		{"primero", 0.0, 1.0 / 3.0},
		{"ultimo", 1.0, 5.0 / 6.0},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Percentile(hist, c.raw); got != c.want {
				t.Fatalf("percentile(%v) = %v, want %v", c.raw, got, c.want)
			}
		})
	}
}

func TestPercentileEmptyHistogram(t *testing.T) {
	if got := Percentile([]int64{}, 0.5); got != 0 {
		t.Fatalf("percentile de histograma vacío = %v, want 0", got)
	}
}
