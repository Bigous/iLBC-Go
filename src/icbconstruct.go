// Port of RFC 3951 Appendix A: iCBConstruct.c.
// Copyright (C) The Internet Society (2004). See LICENSE.

package ilbc

import "math"

func convertEncoderIndices(index span[int]) {
	var k int
	for k = 1; k < 3; k++ {
		if *index.at(k) >= 108 && *index.at(k) < 172 {
			*index.at(k) -= 64
		} else {
			if *index.at(k) >= 236 {
				*index.at(k) -= 128
			} else {
			}
		}
	}
}

func convertDecoderIndices(index span[int]) {
	var k int
	for k = 1; k < 3; k++ {
		if *index.at(k) >= 44 && *index.at(k) < 108 {
			*index.at(k) += 64
		} else {
			if *index.at(k) >= 108 && *index.at(k) < 128 {
				*index.at(k) += 128
			} else {
			}
		}
	}
}

func constructCodebook(decvector span[float32], index span[int], gainIndex span[int], mem span[float32], lMem int, veclen int, nStages int) {
	var j int
	var k int
	var gain [3]float32
	var cbvec [40]float32
	gain[0] = gaindequant(*gainIndex.at(0), float32(1), 32)
	if nStages > 1 {
		gain[1] = gaindequant(*gainIndex.at(1), float32(math.Abs(float64(gain[0]))), 16)
	}
	if nStages > 2 {
		gain[2] = gaindequant(*gainIndex.at(2), float32(math.Abs(float64(gain[1]))), 8)
	}
	getCBvec(span[float32]{data: cbvec[:]}, mem, *index.at(0), lMem, veclen)
	for j = 0; j < veclen; j++ {
		*decvector.at(j) = float32(gain[0] * cbvec[j])
	}
	if nStages > 1 {
		for k = 1; k < nStages; k++ {
			getCBvec(span[float32]{data: cbvec[:]}, mem, *index.at(k), lMem, veclen)
			for j = 0; j < veclen; j++ {
				*decvector.at(j) += float32(gain[k] * cbvec[j])
			}
		}
	}
}
