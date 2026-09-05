package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRender(t *testing.T) {
	out := Render("Hello {{name}} ({{stars}})", map[string]string{
		"name":  "Alex",
		"stars": "5",
	})
	assert.Equal(t, "Hello Alex (5)", out)
	assert.Equal(t, "", Render("", map[string]string{"name": "x"}))
	assert.Equal(t, "keep {{unknown}}", Render("keep {{unknown}}", map[string]string{"name": "x"}))
}
