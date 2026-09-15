// Port of RFC 3951 Appendix A: FrameClassify.c.
// Copyright (C) The Internet Society (2004). See LICENSE.

package ilbc

func frameClassify(encoder *encoderState, residual span[float32]) int {
	var maxSsqEn float32
	var fssqEn [6]float32
	var bssqEn [6]float32
	var pp span[float32]
	var n int
	var l int
	var maxSsqEnN int
	var ssqEnWin [5]float32
	ssqEnWin = [5]float32{float32(0.8), float32(0.9), float32(1), float32(0.9), float32(0.8)}
	var sampEnWin [5]float32
	sampEnWin = [5]float32{float32(0.16666667), float32(0.33333334), float32(0.5), float32(0.6666667), float32(0.8333333)}
	clear(fssqEn[0:6])
	clear(bssqEn[0:6])
	n = 0
	pp = residual
	for l = 0; l < 5; l++ {
		fssqEn[n] += float32(float32(sampEnWin[l]**pp.at(0)) * *pp.at(0))
		pp = pp.add(1)
	}
	for l = 5; l < 40; l++ {
		fssqEn[n] += float32(*pp.at(0) * *pp.at(0))
		pp = pp.add(1)
	}
	for n = 1; n < encoder.subframes-1; n++ {
		pp = residual.add(n * 40)
		for l = 0; l < 5; l++ {
			fssqEn[n] += float32(float32(sampEnWin[l]**pp.at(0)) * *pp.at(0))
			bssqEn[n] += float32(*pp.at(0) * *pp.at(0))
			pp = pp.add(1)
		}
		for l = 5; l < 35; l++ {
			fssqEn[n] += float32(*pp.at(0) * *pp.at(0))
			bssqEn[n] += float32(*pp.at(0) * *pp.at(0))
			pp = pp.add(1)
		}
		for l = 35; l < 40; l++ {
			fssqEn[n] += float32(*pp.at(0) * *pp.at(0))
			bssqEn[n] += float32(float32(sampEnWin[40-l-1]**pp.at(0)) * *pp.at(0))
			pp = pp.add(1)
		}
	}
	n = encoder.subframes - 1
	pp = residual.add(n * 40)
	for l = 0; l < 35; l++ {
		bssqEn[n] += float32(*pp.at(0) * *pp.at(0))
		pp = pp.add(1)
	}
	for l = 35; l < 40; l++ {
		bssqEn[n] += float32(float32(sampEnWin[40-l-1]**pp.at(0)) * *pp.at(0))
		pp = pp.add(1)
	}
	if encoder.mode == 20 {
		l = 1
	} else {
		l = 0
	}
	maxSsqEn = float32((fssqEn[0] + bssqEn[1]) * ssqEnWin[l])
	maxSsqEnN = 1
	for n = 2; n < encoder.subframes; n++ {
		l++
		if float32((fssqEn[n-1]+bssqEn[n])*ssqEnWin[l]) > maxSsqEn {
			maxSsqEn = float32((fssqEn[n-1] + bssqEn[n]) * ssqEnWin[l])
			maxSsqEnN = n
		}
	}
	return maxSsqEnN
}
