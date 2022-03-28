package tools

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestContains(t *testing.T) {
	assert.True(t, Contains([]string{"aaa", "bbb", "ccc"}, "bbb"))
	assert.False(t, Contains([]string{"aaa", "bbb", "ccc"}, "ddd"))
}
