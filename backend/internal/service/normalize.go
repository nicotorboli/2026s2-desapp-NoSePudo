package service

import (
	"strings"
	"unicode"
)

// accentFold traduce los caracteres acentuados más comunes a su forma base
// para que el matcheo entre el catálogo y WhoScored no dependa de diacríticos.
var accentFold = map[rune]rune{
	'á': 'a', 'à': 'a', 'â': 'a', 'ä': 'a', 'ã': 'a', 'å': 'a',
	'é': 'e', 'è': 'e', 'ê': 'e', 'ë': 'e',
	'í': 'i', 'ì': 'i', 'î': 'i', 'ï': 'i',
	'ó': 'o', 'ò': 'o', 'ô': 'o', 'ö': 'o', 'õ': 'o', 'ø': 'o',
	'ú': 'u', 'ù': 'u', 'û': 'u', 'ü': 'u',
	'ñ': 'n', 'ç': 'c', 'ý': 'y', 'ÿ': 'y',
	'š': 's', 'ž': 'z', 'č': 'c', 'ć': 'c', 'đ': 'd', 'ð': 'd',
	'ł': 'l', 'ř': 'r', 'ğ': 'g', 'ş': 's', 'ß': 's',
	'ă': 'a', 'ą': 'a', 'ę': 'e', 'ě': 'e', 'ś': 's', 'ź': 'z', 'ż': 'z',
}

func normalizeName(raw string) string {
	lower := strings.ToLower(strings.TrimSpace(raw))
	var b strings.Builder
	b.Grow(len(lower))
	for _, r := range lower {
		if folded, ok := accentFold[r]; ok {
			r = folded
		}
		b.WriteRune(r)
	}
	return b.String()
}

// clubSuffixes son los sufijos institucionales que football-data agrega al
// nombre del club y que WhoScored omite.
var clubSuffixes = map[string]bool{
	"fc": true, "cf": true, "afc": true, "ac": true, "sc": true,
	"as": true, "ss": true, "sa": true, "rcd": true, "ca": true,
}

// clubAliases normaliza las abreviaturas y traducciones que WhoScored usa
// para algunos clubes. Los casos sin alias se resuelven por el fallback de
// nombre único.
var clubAliases = map[string]string{
	"man city":         "manchester city",
	"man utd":          "manchester united",
	"bayern munchen":   "bayern munich",
	"koln":             "cologne",
	"mgladbach":        "monchengladbach",
	"internazionale":   "inter",
	"paris st germain": "paris saint-germain",
}

// normalizeClub deja el club comparable entre proveedores: minúsculas, sin
// diacríticos, sin puntuación, sin sufijos institucionales y con los alias
// conocidos aplicados.
func normalizeClub(raw string) string {
	lower := normalizeName(raw)
	lower = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			return r
		}
		return -1
	}, lower)

	if alias, ok := clubAliases[strings.Join(strings.Fields(lower), " ")]; ok {
		lower = alias
	}

	words := strings.Fields(lower)
	cleaned := make([]string, 0, len(words))
	for _, word := range words {
		if clubSuffixes[word] {
			continue
		}
		cleaned = append(cleaned, word)
	}
	return strings.Join(cleaned, " ")
}
