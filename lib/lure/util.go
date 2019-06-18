package lure

import (
	"bytes"
	"text/template"
)

// Tprintf passed template string is formatted using its operands and returns the resulting string.
// Spaces are added between operands when neither is a string.
// https://play.golang.org/p/COHKlB2RML
func Tprintf(tmpl string, data map[string]interface{}) string {
	t := template.Must(template.New(tmpl).Parse(tmpl))
	buf := &bytes.Buffer{}
	if err := t.Execute(buf, data); err != nil {
		return ""
	}
	return buf.String()
}
