package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMd5(t *testing.T) {
	assert.Equal(t, "3e25960a79dbc69b674cd4ec67a72c62", Md5("Hello world"))
}
