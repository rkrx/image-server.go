package imaging

func CalcProportionalResize(imgWidth, imgHeight, destWidth, destHeight int) (int, int) {
	if imgWidth > imgHeight {
		resWidth := destWidth
		resHeight := imgHeight * destWidth / imgWidth
		if resHeight > destHeight {
			return imgWidth * destHeight / imgHeight, destHeight
		}
		return resWidth, resHeight
	}
	resWidth := imgWidth * destHeight / imgHeight
	resHeight := destHeight
	if resWidth > destWidth {
		return destWidth, imgHeight * destWidth / imgWidth
	}
	return resWidth, resHeight
}
