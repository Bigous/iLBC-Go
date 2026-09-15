// Port of RFC 3951 Appendix A: StateSearchW.c.
// Copyright (C) The Internet Society (2004). See LICENSE.

package ilbc

import "math"

func absQuantW(encoder *encoderState, input span[float32], syntDenum span[float32], weightDenum span[float32], out span[int], len int, stateFirst int) {
	var syntOut span[float32]
	var syntOutBuf [68]float32
	var toQ float32
	var xq [1]float32
	var n int
	var index [1]int
	clear(syntOutBuf[0:10])
	syntOut = span[float32]{data: syntOutBuf[:], off: 10}
	if stateFirst != 0 {
		allPoleFilter(input, weightDenum, 40, 10)
	} else {
		allPoleFilter(input, weightDenum, encoder.shortStateSamples-40, 10)
	}
	for n = 0; n < len; n++ {
		if stateFirst != 0 && n == 40 {
			syntDenum = syntDenum.add(11)
			weightDenum = weightDenum.add(11)
			allPoleFilter(input.add(n), weightDenum, len-n, 10)
		} else {
			if stateFirst == 0 && n == encoder.shortStateSamples-40 {
				syntDenum = syntDenum.add(11)
				weightDenum = weightDenum.add(11)
				allPoleFilter(input.add(n), weightDenum, len-n, 10)
			}
		}
		*syntOut.at(n) = float32(0)
		allPoleFilter(syntOut.add(n), weightDenum, 1, 10)
		toQ = *input.at(n) - *syntOut.at(n)
		sortSq(span[float32]{data: xq[:]}, span[int]{data: index[:]}, toQ, span[float32]{data: stateSq3Tbl[:]}, 8)
		*out.at(n) = index[0]
		*syntOut.at(n) = stateSq3Tbl[*out.at(n)]
		allPoleFilter(syntOut.add(n), weightDenum, 1, 10)
	}
}

func stateSearchW(encoder *encoderState, residual span[float32], syntDenum span[float32], weightDenum span[float32], idxForMax span[int], idxVec span[int], len int, stateFirst int) {
	var dtmp [1]float32
	var maxVal float32
	var tmpbuf [126]float32
	var tmp span[float32]
	var numerator [11]float32
	var foutbuf [126]float32
	var fout span[float32]
	var k int
	var qmax float32
	var scal float32
	clear(tmpbuf[0:10])
	clear(foutbuf[0:10])
	for k = 0; k < 10; k++ {
		numerator[k] = *syntDenum.at(10 - k)
	}
	numerator[10] = *syntDenum.at(0)
	tmp = span[float32]{data: tmpbuf[:], off: 10}
	fout = span[float32]{data: foutbuf[:], off: 10}
	copy(tmp.slice(len), residual.slice(len))
	clear(tmp.add(len).slice(len))
	zeroPoleFilter(tmp, span[float32]{data: numerator[:]}, syntDenum, 2*len, 10, fout)
	for k = 0; k < len; k++ {
		*fout.at(k) += *fout.at(k + len)
	}
	maxVal = *fout.at(0)
	for k = 1; k < len; k++ {
		if float32(*fout.at(k)**fout.at(k)) > float32(maxVal*maxVal) {
			maxVal = *fout.at(k)
		}
	}
	maxVal = float32(math.Abs(float64(maxVal)))
	if float64(maxVal) < float64(10) {
		maxVal = float32(10)
	}
	maxVal = float32(math.Log10(float64(maxVal)))
	sortSq(span[float32]{data: dtmp[:]}, idxForMax, maxVal, span[float32]{data: stateFrgqTbl[:]}, 64)
	maxVal = stateFrgqTbl[*idxForMax.at(0)]
	qmax = float32(math.Pow(float64(10), float64(maxVal)))
	scal = float32(4.5) / qmax
	for k = 0; k < len; k++ {
		*fout.at(k) *= scal
	}
	absQuantW(encoder, fout, syntDenum, weightDenum, idxVec, len, stateFirst)
}
