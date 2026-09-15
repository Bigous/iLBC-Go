// Port of RFC 3951 Appendix A: filter.c.
// Copyright (C) The Internet Society (2004). See LICENSE.

package ilbc

func allPoleFilter(inOut span[float32], coef span[float32], lengthInOut int, orderCoef int) {
	var n int
	var k int
	for n = 0; n < lengthInOut; n++ {
		for k = 1; k <= orderCoef; k++ {
			*inOut.at(0) -= float32(*coef.at(k) * *inOut.at(-k))
		}
		inOut = inOut.add(1)
	}
}

func allZeroFilter(in span[float32], coef span[float32], lengthInOut int, orderCoef int, out span[float32]) {
	var n int
	var k int
	for n = 0; n < lengthInOut; n++ {
		*out.at(0) = float32(*coef.at(0) * *in.at(0))
		for k = 1; k <= orderCoef; k++ {
			*out.at(0) += float32(*coef.at(k) * *in.at(-k))
		}
		out = out.add(1)
		in = in.add(1)
	}
}

func zeroPoleFilter(in span[float32], zeroCoef span[float32], poleCoef span[float32], lengthInOut int, orderCoef int, out span[float32]) {
	allZeroFilter(in, zeroCoef, lengthInOut, orderCoef, out)
	allPoleFilter(out, poleCoef, lengthInOut, orderCoef)
}

func downSample(in span[float32], coef span[float32], lengthIn int, state span[float32], out span[float32]) {
	var o float32
	var outPtr span[float32]
	outPtr = out
	var coefPtr span[float32]
	var inPtr span[float32]
	var statePtr span[float32]
	var i int
	var j int
	var stop int
	for i = 3; i < lengthIn; i += 2 {
		coefPtr = coef.add(0)
		inPtr = in.add(i)
		statePtr = state.add(5)
		o = float32(0)
		stop = func() int {
			if i < 7 {
				return i + 1
			}
			return 7
		}()
		for j = 0; j < stop; j++ {
			o += float32(*coefPtr.at(0) * *inPtr.at(0))
			coefPtr = coefPtr.add(1)
			inPtr = inPtr.add(-1)
		}
		for j = i + 1; j < 7; j++ {
			o += float32(*coefPtr.at(0) * *statePtr.at(0))
			coefPtr = coefPtr.add(1)
			statePtr = statePtr.add(-1)
		}
		*outPtr.at(0) = o
		outPtr = outPtr.add(1)
	}
	for i = lengthIn + 2; i < lengthIn+3; i += 2 {
		o = float32(0)
		{ // The tail starts at lengthIn+2, so only zero extension is possible.

			coefPtr = coef.add(i - lengthIn)
			inPtr = in.add(lengthIn - 1)
			for j = 0; j < 7-(i-lengthIn); j++ {
				o += float32(*coefPtr.at(0) * *inPtr.at(0))
				coefPtr = coefPtr.add(1)
				inPtr = inPtr.add(-1)
			}
		}
		*outPtr.at(0) = o
		outPtr = outPtr.add(1)
	}
}
