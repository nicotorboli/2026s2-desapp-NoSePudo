package service

import "strings"

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
