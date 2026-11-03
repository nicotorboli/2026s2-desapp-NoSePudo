package model

const ScoreHistogramBuckets = 1000

func BuildHistogram(scores []float64, buckets int) []int64 {
	hist := make([]int64, buckets)
	for _, score := range scores {
		hist[bucketIndex(score, buckets)]++
	}
	return hist
}

func Percentile(hist []int64, raw float64) float64 {
	buckets := len(hist)
	if buckets == 0 {
		return 0
	}

	idx := bucketIndex(raw, buckets)

	var below, total int64
	for i, count := range hist {
		total += count
		if i < idx {
			below += count
		}
	}
	if total == 0 {
		return 0
	}

	return (float64(below) + 0.5*float64(hist[idx])) / float64(total)
}

func bucketIndex(score float64, buckets int) int {
	idx := int(score * float64(buckets))
	if idx < 0 {
		return 0
	}
	if idx >= buckets {
		return buckets - 1
	}
	return idx
}
