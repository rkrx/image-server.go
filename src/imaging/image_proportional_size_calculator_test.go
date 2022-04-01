package imaging

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCalcProportionalSize(t *testing.T) {
	var w, h int

	w, h = CalcProportionalResize(300, 200, 450, 300)
	assert.Equal(t, 450, w)
	assert.Equal(t, 300, h)

	w, h = CalcProportionalResize(400, 200, 300, 300)
	assert.Equal(t, 300, w)
	assert.Equal(t, 150, h)

	w, h = CalcProportionalResize(300, 300, 450, 300)
	assert.Equal(t, 300, w)
	assert.Equal(t, 300, h)
}

func TestCalcProportionalSize_RealWorld(t *testing.T) {
	var w, h int

	w, h = CalcProportionalResize(2119, 2120, 300, 300)
	assert.Equal(t, 299, w)
	assert.Equal(t, 300, h)
}
