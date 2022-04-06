package imaging

import "math"

func CalcProportionalResize(imgWidth, imgHeight, destWidth, destHeight int) (int, int) {
	srcW, srcH, dstW, dstH := float64(imgWidth), float64(imgHeight), float64(destWidth), float64(destHeight)

	roundedInt := func(val float64) int {
		return int(math.Round(val))
	}

	if dstH == 0 && dstW > 0 {
		dstH = dstW / srcW * srcH
	} else if dstW == 0 && dstH > 0 {
		dstW = dstH / srcH * srcW
	} else if dstW == 0 && dstH == 0 {
		dstW, dstH = srcW, srcH
	}

	if srcW > srcH {
		resWidth := dstW
		resHeight := srcH * dstW / srcW
		if resHeight > dstH {
			return roundedInt(srcW * dstH / srcH), roundedInt(dstH)
		}
		return roundedInt(resWidth), roundedInt(resHeight)
	}

	resWidth := srcW * dstH / srcH
	resHeight := dstH
	if resWidth > dstW {
		return roundedInt(dstW), roundedInt(srcH * dstW / srcW)
	}

	return roundedInt(resWidth), roundedInt(resHeight)
}
