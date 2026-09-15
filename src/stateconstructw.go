// Port of RFC 3951 Appendix A: StateConstructW.c.
// Copyright (C) The Internet Society (2004). See LICENSE.

package ilbc

import "math"

func stateConstructW(idxForMax int, idxVec span[int], syntDenum span[float32], out span[float32], len int) {
	var maxVal float32
	var tmpbuf [170]float32
	var tmp span[float32]
	var numerator [11]float32
	var foutbuf [170]float32
	var fout span[float32]
	var k int
	var tmpi int
	maxVal = stateFrgqTbl[idxForMax]
	maxVal = float32(math.Pow(float64(10), float64(maxVal))) / float32(4.5)
	clear(tmpbuf[0:10])
	clear(foutbuf[0:10])
	for k = 0; k < 10; k++ {
		numerator[k] = *syntDenum.at(10 - k)
	}
	numerator[10] = *syntDenum.at(0)
	tmp = span[float32]{data: tmpbuf[:], off: 10}
	fout = span[float32]{data: foutbuf[:], off: 10}
	for k = 0; k < len; k++ {
		tmpi = len - 1 - k
		*tmp.at(k) = float32(maxVal * stateSq3Tbl[*idxVec.at(tmpi)])
	}
	clear(tmp.add(len).slice(len))
	zeroPoleFilter(tmp, span[float32]{data: numerator[:]}, syntDenum, 2*len, 10, fout)
	for k = 0; k < len; k++ {
		*out.at(k) = *fout.at(len - 1 - k) + *fout.at(2*len - 1 - k)
	}
}
