// Package template renders {{placeholder}} notification message bodies.
package template

import "strings"

// Render replaces {{placeholders}} in tmpl. Unknown placeholders are left unchanged.
func Render(tmpl string, vals map[string]string) string {
	if tmpl == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(tmpl))
	for i := 0; i < len(tmpl); {
		if next, ok := writePlaceholder(tmpl, i, vals, &b); ok {
			i = next
			continue
		}
		b.WriteByte(tmpl[i])
		i++
	}
	return b.String()
}

func writePlaceholder(tmpl string, i int, vals map[string]string, b *strings.Builder) (next int, ok bool) {
	key, end, found := placeholderKey(tmpl, i)
	if !found {
		return 0, false
	}
	v, hasVal := vals[key]
	if !hasVal {
		return 0, false
	}
	b.WriteString(v)
	return i + 2 + end + 2, true
}

func placeholderKey(tmpl string, i int) (key string, end int, ok bool) {
	if tmpl[i] != '{' || i+1 >= len(tmpl) || tmpl[i+1] != '{' {
		return "", 0, false
	}
	end = strings.Index(tmpl[i+2:], "}}")
	if end < 0 {
		return "", 0, false
	}
	return strings.TrimSpace(tmpl[i+2 : i+2+end]), end, true
}
