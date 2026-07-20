package generator

import "strings"

var irregularPlurals = map[string]string{
	"Person": "People",
}

func pluralize(name string) string {
	if plural, ok := irregularPlurals[name]; ok {
		return plural
	}
	if strings.HasSuffix(name, "s") || strings.HasSuffix(name, "x") || strings.HasSuffix(name, "z") || strings.HasSuffix(name, "ch") || strings.HasSuffix(name, "sh") {
		return name + "es"
	}
	if strings.HasSuffix(name, "y") && len(name) > 1 && !isVowel(name[len(name)-2]) {
		return name[:len(name)-1] + "ies"
	}
	return name + "s"
}

func routeName(name string) string {
	return strings.ToLower(name)
}

func isVowel(letter byte) bool {
	switch letter {
	case 'a', 'e', 'i', 'o', 'u', 'A', 'E', 'I', 'O', 'U':
		return true
	default:
		return false
	}
}
