package imaging

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCalcProportionalContainSize(t *testing.T) {
	var w, h int

	w, h = CalcProportionalInnerMeasures(300, 200, 450, 300)
	assert.Equal(t, 450, w)
	assert.Equal(t, 300, h)

	w, h = CalcProportionalInnerMeasures(400, 200, 300, 300)
	assert.Equal(t, 300, w)
	assert.Equal(t, 150, h)

	w, h = CalcProportionalInnerMeasures(300, 300, 450, 300)
	assert.Equal(t, 300, w)
	assert.Equal(t, 300, h)

	w, h = CalcProportionalInnerMeasures(300, 300, 0, 500)
	assert.Equal(t, 500, w)
	assert.Equal(t, 500, h)

	w, h = CalcProportionalInnerMeasures(300, 300, 500, 0)
	assert.Equal(t, 500, w)
	assert.Equal(t, 500, h)

	w, h = CalcProportionalInnerMeasures(300, 300, 0, 0)
	assert.Equal(t, 300, w)
	assert.Equal(t, 300, h)
}

func TestCalcProportionalCoverSize(t *testing.T) {
	var w, h int

	w, h = CalcProportionalOuterMeasures(100, 100, 200, 200)
	assert.Equal(t, 200, w)
	assert.Equal(t, 200, h)

	w, h = CalcProportionalOuterMeasures(100, 200, 200, 200)
	assert.Equal(t, 200, w)
	assert.Equal(t, 400, h)

	w, h = CalcProportionalOuterMeasures(400, 200, 200, 100)
	assert.Equal(t, 200, w)
	assert.Equal(t, 100, h)

	w, h = CalcProportionalOuterMeasures(400, 200, 100, 200)
	assert.Equal(t, 400, w)
	assert.Equal(t, 200, h)

	w, h = CalcProportionalOuterMeasures(200, 400, 200, 100)
	assert.Equal(t, 200, w)
	assert.Equal(t, 400, h)

	w, h = CalcProportionalOuterMeasures(200, 400, 100, 200)
	assert.Equal(t, 100, w)
	assert.Equal(t, 200, h)

	w, h = CalcProportionalOuterMeasures(200, 400, 0, 500)
	assert.Equal(t, 250, w)
	assert.Equal(t, 500, h)

	w, h = CalcProportionalOuterMeasures(300, 300, 500, 0)
	assert.Equal(t, 500, w)
	assert.Equal(t, 500, h)

	w, h = CalcProportionalOuterMeasures(300, 300, 0, 0)
	assert.Equal(t, 300, w)
	assert.Equal(t, 300, h)
}

func TestCalcProportionalSize_RealWorld(t *testing.T) {
	var w, h int

	w, h = CalcProportionalInnerMeasures(2119, 2120, 300, 300)
	assert.Equal(t, 300, w)
	assert.Equal(t, 300, h)
}
