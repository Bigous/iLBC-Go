// Port of RFC 3951 Appendix A: LPCdecode.c.
// Copyright (C) The Internet Society (2004). See LICENSE.

package ilbc

func interpolateDecoderLSF(a span[float32], lsf1 span[float32], lsf2 span[float32], coef float32, length int) {
	var lsftmp [10]float32
	interpolate(span[float32]{data: lsftmp[:]}, lsf1, lsf2, coef, length)
	lsf2a(a, span[float32]{data: lsftmp[:]})
}

func dequantizeLSF(lsfdeq span[float32], index span[int], lpcAnalyses int) {
	var i int
	var j int
	var pos int
	var cbPos int
	pos = 0
	cbPos = 0
	for i = 0; i < 3; i++ {
		for j = 0; j < dimLsfCbTbl[i]; j++ {
			*lsfdeq.at(pos + j) = lsfCbTbl[cbPos+int(int32(*index.at(i)))*dimLsfCbTbl[i]+j]
		}
		pos += dimLsfCbTbl[i]
		cbPos += sizeLsfCbTbl[i] * dimLsfCbTbl[i]
	}
	if lpcAnalyses > 1 {
		pos = 0
		cbPos = 0
		for i = 0; i < 3; i++ {
			for j = 0; j < dimLsfCbTbl[i]; j++ {
				*lsfdeq.at(10 + pos + j) = lsfCbTbl[cbPos+int(int32(*index.at(3 + i)))*dimLsfCbTbl[i]+j]
			}
			pos += dimLsfCbTbl[i]
			cbPos += sizeLsfCbTbl[i] * dimLsfCbTbl[i]
		}
	}
}

func decoderInterpolateLSF(syntdenum span[float32], weightdenum span[float32], lsfdeq span[float32], length int, decoder *decoderState) {
	var i int
	var pos int
	var lpLength int
	var lp [11]float32
	var lsfdeq2 span[float32]
	lsfdeq2 = lsfdeq.add(length)
	lpLength = length + 1
	if decoder.mode == 30 {
		interpolateDecoderLSF(span[float32]{data: lp[:]}, span[float32]{data: decoder.lsfdeqold[:]}, lsfdeq, lsfWeightTbl30ms[0], length)
		copy(syntdenum.slice(lpLength), lp[0:lpLength])
		bwexpand(weightdenum, span[float32]{data: lp[:]}, float32(0.4222), lpLength)
		pos = lpLength
		for i = 1; i < 6; i++ {
			interpolateDecoderLSF(span[float32]{data: lp[:]}, lsfdeq, lsfdeq2, lsfWeightTbl30ms[i], length)
			copy(syntdenum.add(pos).slice(lpLength), lp[0:lpLength])
			bwexpand(weightdenum.add(pos), span[float32]{data: lp[:]}, float32(0.4222), lpLength)
			pos += lpLength
		}
	} else {
		pos = 0
		for i = 0; i < decoder.subframes; i++ {
			interpolateDecoderLSF(span[float32]{data: lp[:]}, span[float32]{data: decoder.lsfdeqold[:]}, lsfdeq, lsfWeightTbl20ms[i], length)
			copy(syntdenum.add(pos).slice(lpLength), lp[0:lpLength])
			bwexpand(weightdenum.add(pos), span[float32]{data: lp[:]}, float32(0.4222), lpLength)
			pos += lpLength
		}
	}
	if decoder.mode == 30 {
		copy(decoder.lsfdeqold[0:length], lsfdeq2.slice(length))
	} else {
		copy(decoder.lsfdeqold[0:length], lsfdeq.slice(length))
	}
}
