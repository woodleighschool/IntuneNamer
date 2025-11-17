package naming

import (
	"regexp"
	"strings"
	"text/template"
	"unicode"
)

var whitespaceRegex = regexp.MustCompile(`\s+`)

func templateFuncMap() template.FuncMap {
	return template.FuncMap{
		"upper":    strings.ToUpper,
		"lower":    strings.ToLower,
		"title":    titleCase,
		"default":  defaultString,
		"replace":  strings.ReplaceAll,
		"truncate": truncate,
		"substr":   substr,
		"clean":    cleanString,
	}
}

func titleCase(input string) string {
	words := strings.Fields(strings.ToLower(input))
	for i, w := range words {
		if len(w) == 0 {
			continue
		}
		runes := []rune(w)
		runes[0] = unicode.ToUpper(runes[0])
		words[i] = string(runes)
	}
	return strings.Join(words, " ")
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func truncate(length int, input string) string {
	if length <= 0 {
		return input
	}
	runes := []rune(input)
	if len(runes) <= length {
		return input
	}
	return string(runes[:length])
}

func substr(start, length int, input string) string {
	if length <= 0 {
		return ""
	}
	runes := []rune(input)
	if start >= len(runes) {
		return ""
	}
	end := start + length
	if end > len(runes) {
		end = len(runes)
	}
	return string(runes[start:end])
}

func cleanString(input string) string {
	input = whitespaceRegex.ReplaceAllString(strings.TrimSpace(input), "-")
	var b strings.Builder
	for _, r := range input {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r):
			b.WriteRune(unicode.ToUpper(r))
		case r == '-', r == '_':
			if b.Len() == 0 || lastRune(&b) == '-' {
				continue
			}
			b.WriteRune('-')
		default:
		}
	}
	return b.String()
}

func lastRune(b *strings.Builder) rune {
	str := b.String()
	if str == "" {
		return 0
	}
	runes := []rune(str)
	return runes[len(runes)-1]
}
