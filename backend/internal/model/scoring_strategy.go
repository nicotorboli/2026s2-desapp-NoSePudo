package model

const (
	goalsMax         = 30.0
	assistsMax       = 20.0
	shotsMax         = 4.0
	keyPassesMax     = 3.0
	dribblesMax      = 4.0
	tacklesMax       = 4.0
	interceptionsMax = 3.0
	foulsMax         = 3.0
	clearsMax        = 6.0
	blocksMax        = 2.0
	wasDribbledMax   = 2.0
)

const UnknownPositionDiscount = 0.5

var knownPositions = map[string]bool{
	"Goalkeeper": true,
	"Defender":   true,
	"Midfielder": true,
	"Attacker":   true,
}

func PositionDiscount(position string) float64 {
	if knownPositions[position] {
		return 1.0
	}
	return UnknownPositionDiscount
}

var allStats = []string{
	"goals", "assists", "shots", "key_passes", "dribbles", "tackles",
	"interceptions", "fouls", "clears", "blocks", "was_dribbled",
}

type ScoringStrategy struct {
	Weights map[string]map[string]float64
	Name    string
	Alpha   float64
	ID      int
}

func (s ScoringStrategy) Score(position string, stats PlayerStatistics) float64 {
	w := s.Weights[position]
	if w == nil {
		return s.scoreFallback(stats)
	}
	acc := 0.0
	for stat, weight := range w {
		acc += weight * normalizeStat(stat, stats)
	}
	return s.Alpha*(stats.Rating/10.0) + (1-s.Alpha)*acc
}

func (s ScoringStrategy) scoreFallback(stats PlayerStatistics) float64 {
	acc := 0.0
	for _, stat := range allStats {
		acc += normalizeStat(stat, stats)
	}
	acc /= float64(len(allStats))
	return s.Alpha*(stats.Rating/10.0) + (1-s.Alpha)*acc
}

func normalizeStat(stat string, stats PlayerStatistics) float64 {
	var value, max float64
	invert := false
	switch stat {
	case "goals":
		value, max = float64(stats.Goals), goalsMax
	case "assists":
		value, max = float64(stats.Assists), assistsMax
	case "shots":
		value, max = stats.Shots, shotsMax
	case "key_passes":
		value, max = stats.KeyPasses, keyPassesMax
	case "dribbles":
		value, max = stats.Dribbles, dribblesMax
	case "tackles":
		value, max = stats.Tackles, tacklesMax
	case "interceptions":
		value, max = stats.Interceptions, interceptionsMax
	case "fouls":
		value, max, invert = stats.Fouls, foulsMax, true
	case "clears":
		value, max = stats.Clears, clearsMax
	case "blocks":
		value, max = stats.Blocks, blocksMax
	case "was_dribbled":
		value, max, invert = stats.WasDribbled, wasDribbledMax, true
	default:
		return 0
	}
	n := value / max
	if n > 1 {
		n = 1
	}
	if n < 0 {
		n = 0
	}
	if invert {
		n = 1 - n
	}
	return n
}
