package tag

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPersonAndJoin(t *testing.T) {
	const prefix = "starapp_uid_"
	assert.Equal(t, "starapp_uid_42", Person(prefix, 42))
	assert.Equal(t, "starapp_uid_1,starapp_uid_3", Join(prefix, []int{1, 3}))
	assert.Empty(t, Join(prefix, nil))
}
