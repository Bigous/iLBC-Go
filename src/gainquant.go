// Port of RFC 3951 Appendix A: gainquant.c.
// Copyright (C) The Internet Society (2004). See LICENSE.

package ilbc

import "math"

func gainquant(input float32, maxIn float32, cblen int, index span[int]) float32 {
	var i int
	var tindex int
	var minmeasure float32
	var measure float32
	var cb span[float32]
	var scale float32
	scale = maxIn
	if float64(scale) < float64(0.1) {
		scale = float32(0.1)
	}
	if cblen == 8 {
		cb = span[float32]{data: gainSq3Tbl[:]}
	} else {
		if cblen == 16 {
			cb = span[float32]{data: gainSq4Tbl[:]}
		} else {
			cb = span[float32]{data: gainSq5Tbl[:]}
		}
	}
	minmeasure = float32(1e+07)
	tindex = 0
	for i = 0; i < cblen; i++ {
		measure = float32((input - float32(scale**cb.at(i))) * (input - float32(scale**cb.at(i))))
		if measure < minmeasure {
			tindex = i
			minmeasure = measure
		}
	}
	*index.at(0) = tindex
	return float32(scale * *cb.at(tindex))
}

func gaindequant(index int, maxIn float32, cblen int) float32 {
	var scale float32
	scale = float32(math.Abs(float64(maxIn)))
	if float64(scale) < float64(0.1) {
		scale = float32(0.1)
	}
	if cblen == 8 {
		return float32(scale * gainSq3Tbl[index])
	} else {
		if cblen == 16 {
			return float32(scale * gainSq4Tbl[index])
		} else {
			if cblen == 32 {
				return float32(scale * gainSq5Tbl[index])
			}
		}
	}
	return float32(0)
}
