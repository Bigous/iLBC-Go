// Port of RFC 3951 Appendix A: LPCencode.c.
// Copyright (C) The Internet Society (2004). See LICENSE.

package ilbc

func simpleAnalysis(lsf span[float32], data span[float32], encoder *encoderState) {
	var k int
	var is int
	var temp [240]float32
	var lp [11]float32
	var lp2 [11]float32
	var r [11]float32
	is = 300 - encoder.frameSamples
	copy(encoder.lpcBuffer[is:is+encoder.frameSamples], data.slice(encoder.frameSamples))
	for k = 0; k < encoder.lpcAnalyses; k++ {
		is = 60
		if k < encoder.lpcAnalyses-1 {
			window(span[float32]{data: temp[:]}, span[float32]{data: lpcWinTbl[:]}, span[float32]{data: encoder.lpcBuffer[:]}, 240)
		} else {
			window(span[float32]{data: temp[:]}, span[float32]{data: lpcAsymwinTbl[:]}, span[float32]{data: encoder.lpcBuffer[:], off: is}, 240)
		}
		autocorr(span[float32]{data: r[:]}, span[float32]{data: temp[:]}, 240, 10)
		window(span[float32]{data: r[:]}, span[float32]{data: r[:]}, span[float32]{data: lpcLagwinTbl[:]}, 11)
		levdurb(span[float32]{data: lp[:]}, span[float32]{data: temp[:]}, span[float32]{data: r[:]}, 10)
		bwexpand(span[float32]{data: lp2[:]}, span[float32]{data: lp[:]}, float32(0.9025), 11)
		a2lsf(lsf.add(k*10), span[float32]{data: lp2[:]})
	}
	is = 300 - encoder.frameSamples
	copy(encoder.lpcBuffer[0:is], encoder.lpcBuffer[300+-is:300+-is+is])
}

func interpolateEncoderLSF(a span[float32], lsf1 span[float32], lsf2 span[float32], coef float32, length int32) {
	var lsftmp [10]float32
	interpolate(span[float32]{data: lsftmp[:]}, lsf1, lsf2, coef, int(length))
	lsf2a(a, span[float32]{data: lsftmp[:]})
}

func simpleInterpolateLSF(syntdenum span[float32], weightdenum span[float32], lsf span[float32], lsfdeq span[float32], lsfold span[float32], lsfdeqold span[float32], length int, encoder *encoderState) {
	var i int
	var pos int
	var lpLength int
	var lp [11]float32
	var lsf2 span[float32]
	var lsfdeq2 span[float32]
	lsf2 = lsf.add(length)
	lsfdeq2 = lsfdeq.add(length)
	lpLength = length + 1
	if encoder.mode == 30 {
		interpolateEncoderLSF(span[float32]{data: lp[:]}, lsfdeqold, lsfdeq, lsfWeightTbl30ms[0], int32(length))
		copy(syntdenum.slice(lpLength), lp[0:lpLength])
		interpolateEncoderLSF(span[float32]{data: lp[:]}, lsfold, lsf, lsfWeightTbl30ms[0], int32(length))
		bwexpand(weightdenum, span[float32]{data: lp[:]}, float32(0.4222), lpLength)
		pos = lpLength
		for i = 1; i < encoder.subframes; i++ {
			interpolateEncoderLSF(span[float32]{data: lp[:]}, lsfdeq, lsfdeq2, lsfWeightTbl30ms[i], int32(length))
			copy(syntdenum.add(pos).slice(lpLength), lp[0:lpLength])
			interpolateEncoderLSF(span[float32]{data: lp[:]}, lsf, lsf2, lsfWeightTbl30ms[i], int32(length))
			bwexpand(weightdenum.add(pos), span[float32]{data: lp[:]}, float32(0.4222), lpLength)
			pos += lpLength
		}
	} else {
		pos = 0
		for i = 0; i < encoder.subframes; i++ {
			interpolateEncoderLSF(span[float32]{data: lp[:]}, lsfdeqold, lsfdeq, lsfWeightTbl20ms[i], int32(length))
			copy(syntdenum.add(pos).slice(lpLength), lp[0:lpLength])
			interpolateEncoderLSF(span[float32]{data: lp[:]}, lsfold, lsf, lsfWeightTbl20ms[i], int32(length))
			bwexpand(weightdenum.add(pos), span[float32]{data: lp[:]}, float32(0.4222), lpLength)
			pos += lpLength
		}
	}
	if encoder.mode == 30 {
		copy(lsfold.slice(length), lsf2.slice(length))
		copy(lsfdeqold.slice(length), lsfdeq2.slice(length))
	} else {
		copy(lsfold.slice(length), lsf.slice(length))
		copy(lsfdeqold.slice(length), lsfdeq.slice(length))
	}
}

func quantizeLSF(lsfdeq span[float32], index span[int], lsf span[float32], lpcAnalyses int) {
	splitVQ(lsfdeq, index, lsf, span[float32]{data: lsfCbTbl[:]}, 3, span[int]{data: dimLsfCbTbl[:]}, span[int]{data: sizeLsfCbTbl[:]})
	if lpcAnalyses == 2 {
		splitVQ(lsfdeq.add(10), index.add(3), lsf.add(10), span[float32]{data: lsfCbTbl[:]}, 3, span[int]{data: dimLsfCbTbl[:]}, span[int]{data: sizeLsfCbTbl[:]})
	}
}

func encodeLPC(syntdenum span[float32], weightdenum span[float32], lsfIndex span[int], data span[float32], encoder *encoderState) {
	var lsf [20]float32
	var lsfdeq [20]float32
	simpleAnalysis(span[float32]{data: lsf[:]}, data, encoder)
	quantizeLSF(span[float32]{data: lsfdeq[:]}, lsfIndex, span[float32]{data: lsf[:]}, encoder.lpcAnalyses)
	checkLSF(span[float32]{data: lsfdeq[:]}, 10, encoder.lpcAnalyses)
	simpleInterpolateLSF(syntdenum, weightdenum, span[float32]{data: lsf[:]}, span[float32]{data: lsfdeq[:]}, span[float32]{data: encoder.lsfold[:]}, span[float32]{data: encoder.lsfdeqold[:]}, 10, encoder)
}
