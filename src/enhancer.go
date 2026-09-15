// Port of RFC 3951 Appendix A: enhancer.c.
// Copyright (C) The Internet Society (2004). See LICENSE.

package ilbc

import "math"

func nearestNeighbor(index span[int], array span[float32], value float32, arlength int) {
	var i int
	var bestcrit float32
	var crit float32
	crit = *array.at(0) - value
	bestcrit = float32(crit * crit)
	*index.at(0) = 0
	for i = 1; i < arlength; i++ {
		crit = *array.at(i) - value
		crit = float32(crit * crit)
		if crit < bestcrit {
			bestcrit = crit
			*index.at(0) = i
		}
	}
}

func mycorr1(corr span[float32], seq1 span[float32], dim1 int, seq2 span[float32], dim2 int) {
	var i int
	var j int
	for i = 0; i <= dim1-dim2; i++ {
		*corr.at(i) = float32(0)
		for j = 0; j < dim2; j++ {
			*corr.at(i) += float32(*seq1.at(i + j) * *seq2.at(j))
		}
	}
}

func upsampleEnhancement(useq1 span[float32], seq1 span[float32], dim1 int, hfl int) {
	var pu span[float32]
	var ps span[float32]
	var i int
	var j int
	var k int
	var q int
	var filterlength int
	var hfl2 int
	var polyp [4]span[float32]
	var pp span[float32]
	filterlength = 2*hfl + 1
	if filterlength > dim1 {
		hfl2 = dim1 / 2
		for j = 0; j < 4; j++ {
			polyp[j] = span[float32]{data: polyphaserTbl[:], off: j*filterlength + hfl + -hfl2}
		}
		hfl = hfl2
		filterlength = 2*hfl + 1
	} else {
		for j = 0; j < 4; j++ {
			polyp[j] = span[float32]{data: polyphaserTbl[:], off: j * filterlength}
		}
	}
	pu = useq1
	for i = hfl; i < filterlength; i++ {
		for j = 0; j < 4; j++ {
			*pu.at(0) = float32(0)
			pp = polyp[j]
			ps = seq1.add(i)
			for k = 0; k <= i; k++ {
				*pu.at(0) += float32(*ps.at(0) * *pp.at(0))
				ps = ps.add(-1)
				pp = pp.add(1)
			}
			pu = pu.add(1)
		}
	}
	for i = filterlength; i < dim1; i++ {
		for j = 0; j < 4; j++ {
			*pu.at(0) = float32(0)
			pp = polyp[j]
			ps = seq1.add(i)
			for k = 0; k < filterlength; k++ {
				*pu.at(0) += float32(*ps.at(0) * *pp.at(0))
				ps = ps.add(-1)
				pp = pp.add(1)
			}
			pu = pu.add(1)
		}
	}
	for q = 1; q <= hfl; q++ {
		for j = 0; j < 4; j++ {
			*pu.at(0) = float32(0)
			pp = polyp[j].add(q)
			ps = seq1.add(dim1 + -1)
			for k = 0; k < filterlength-q; k++ {
				*pu.at(0) += float32(*ps.at(0) * *pp.at(0))
				ps = ps.add(-1)
				pp = pp.add(1)
			}
			pu = pu.add(1)
		}
	}
}

func refiner(seg span[float32], updStartPos span[float32], idata span[float32], idatal int, centerStartPos int, estSegPos float32, period float32) {
	var estSegPosRounded int
	var searchSegStartPos int
	var searchSegEndPos int
	var corrdim int
	var tloc int
	var tloc2 int
	var i int
	var st int
	var en int
	var fraction int
	var vect [86]float32
	var corrVec [5]float32
	var maxv float32
	var corrVecUps [20]float32
	estSegPosRounded = int(float64(estSegPos) - float64(0.5))
	searchSegStartPos = estSegPosRounded - 2
	if searchSegStartPos < 0 {
		searchSegStartPos = 0
	}
	searchSegEndPos = estSegPosRounded + 2
	if searchSegEndPos+80 >= idatal {
		searchSegEndPos = idatal - 80 - 1
	}
	corrdim = searchSegEndPos - searchSegStartPos + 1
	mycorr1(span[float32]{data: corrVec[:]}, idata.add(searchSegStartPos), corrdim+80-1, idata.add(centerStartPos), 80)
	upsampleEnhancement(span[float32]{data: corrVecUps[:]}, span[float32]{data: corrVec[:]}, corrdim, 3)
	tloc = 0
	maxv = corrVecUps[0]
	for i = 1; i < 4*corrdim; i++ {
		if corrVecUps[i] > maxv {
			tloc = i
			maxv = corrVecUps[i]
		}
	}
	*updStartPos.at(0) = float32(searchSegStartPos) + float32(tloc)/float32(4) + float32(1)
	tloc2 = tloc / 4
	if tloc > tloc2*4 {
		tloc2++
	}
	st = searchSegStartPos + tloc2 - 3
	if st < 0 {
		clear(vect[0:-st])
		copy(vect[-st:-st+(86+st)], idata.slice(86+st))
	} else {
		en = st + 86
		if en > idatal {
			copy(vect[0:86-(en-idatal)], idata.add(st).slice(86-(en-idatal)))
			clear(vect[86-(en-idatal) : 86-(en-idatal)+(en-idatal)])
		} else {
			copy(vect[0:86], idata.add(st).slice(86))
		}
	}
	fraction = tloc2*4 - tloc
	mycorr1(seg, span[float32]{data: vect[:]}, 86, span[float32]{data: polyphaserTbl[:], off: 7 * fraction}, 7)
}

func smath(odata span[float32], sseq span[float32], hl int, alpha0 float32) {
	var i int
	var k int
	var w00 float32
	var w10 float32
	var w11 float32
	var a float32
	var b float32
	var c float32
	var psseq span[float32]
	var err float32
	var errs float32
	var surround [240]float32
	var wt [7]float32
	var denom float32
	for i = 1; i <= 2*hl+1; i++ {
		wt[i-1] = float32(float32(0.5) * (float32(1) - float32(math.Cos(float64(float32(float32(6.2831855)*float32(i))/float32(2*hl+2))))))
	}
	wt[hl] = float32(0)
	for i = 0; i < 80; i++ {
		surround[i] = float32(*sseq.at(i) * wt[0])
	}
	for k = 1; k < hl; k++ {
		psseq = sseq.add(k * 80)
		for i = 0; i < 80; i++ {
			surround[i] += float32(*psseq.at(i) * wt[k])
		}
	}
	for k = hl + 1; k <= 2*hl; k++ {
		psseq = sseq.add(k * 80)
		for i = 0; i < 80; i++ {
			surround[i] += float32(*psseq.at(i) * wt[k])
		}
	}
	w00 = float32(0)
	w11 = float32(0)
	w10 = float32(0)
	psseq = sseq.add(hl * 80)
	for i = 0; i < 80; i++ {
		w00 += float32(*psseq.at(i) * *psseq.at(i))
		w11 += float32(surround[i] * surround[i])
		w10 += float32(surround[i] * *psseq.at(i))
	}
	if math.Abs(float64(w11)) < float64(1) {
		w11 = float32(1)
	}
	c = float32(math.Sqrt(float64(w00 / w11)))
	errs = float32(0)
	psseq = sseq.add(hl * 80)
	for i = 0; i < 80; i++ {
		*odata.at(i) = float32(c * surround[i])
		err = *psseq.at(i) - *odata.at(i)
		errs += float32(err * err)
	}
	if errs > float32(alpha0*w00) {
		if w00 < float32(1) {
			w00 = float32(1)
		}
		denom = (float32(w11*w00) - float32(w10*w10)) / float32(w00*w00)
		if float64(denom) > float64(0.0001) {
			a = float32(math.Sqrt(float64((alpha0 - float32(alpha0*alpha0)/float32(4)) / denom)))
			b = -alpha0/float32(2) - float32(a*w10)/w00
			b = b + float32(1)
		} else {
			a = float32(0)
			b = float32(1)
		}
		psseq = sseq.add(hl * 80)
		for i = 0; i < 80; i++ {
			*odata.at(i) = float32(a*surround[i]) + float32(b**psseq.at(i))
		}
	}
}

func getsseq(sseq span[float32], idata span[float32], idatal int, centerStartPos int, period span[float32], plocs span[float32], periodl int, hl int) {
	var i int
	var centerEndPos int
	var q int
	var blockStartPos [7]float32
	var lagBlock [7]int
	var plocs2 [20]float32
	var psseq span[float32]
	centerEndPos = centerStartPos + 80 - 1
	nearestNeighbor(span[int]{data: lagBlock[:], off: hl}, plocs, float32(float32(0.5)*float32(centerStartPos+centerEndPos)), periodl)
	blockStartPos[hl] = float32(centerStartPos)
	psseq = sseq.add(80 * hl)
	copy(psseq.slice(80), idata.add(centerStartPos).slice(80))
	for q = hl - 1; q >= 0; q-- {
		blockStartPos[q] = blockStartPos[q+1] - *period.at(lagBlock[q+1])
		nearestNeighbor(span[int]{data: lagBlock[:], off: q}, plocs, blockStartPos[q]+float32(40)-*period.at(lagBlock[q+1]), periodl)
		if blockStartPos[q]-float32(2) >= float32(0) {
			refiner(sseq.add(q*80), span[float32]{data: blockStartPos[:], off: q}, idata, idatal, centerStartPos, blockStartPos[q], *period.at(lagBlock[q+1]))
		} else {
			psseq = sseq.add(q * 80)
			clear(psseq.slice(80))
		}
	}
	for i = 0; i < periodl; i++ {
		plocs2[i] = *plocs.at(i) - *period.at(i)
	}
	for q = hl + 1; q <= 2*hl; q++ {
		nearestNeighbor(span[int]{data: lagBlock[:], off: q}, span[float32]{data: plocs2[:]}, blockStartPos[q-1]+float32(40), periodl)
		blockStartPos[q] = blockStartPos[q-1] + *period.at(lagBlock[q])
		if blockStartPos[q]+float32(80)+float32(2) < float32(idatal) {
			refiner(sseq.add(80*q), span[float32]{data: blockStartPos[:], off: q}, idata, idatal, centerStartPos, blockStartPos[q], *period.at(lagBlock[q]))
		} else {
			psseq = sseq.add(q * 80)
			clear(psseq.slice(80))
		}
	}
}

func enhancer(odata span[float32], idata span[float32], idatal int, centerStartPos int, alpha0 float32, period span[float32], plocs span[float32], periodl int) {
	var sseq [560]float32
	getsseq(span[float32]{data: sseq[:]}, idata, idatal, centerStartPos, period, plocs, periodl, 3)
	smath(odata, span[float32]{data: sseq[:]}, 3, alpha0)
}

func xCorrCoef(target span[float32], regressor span[float32], subl int) float32 {
	var i int
	var ftmp1 float32
	var ftmp2 float32
	ftmp1 = float32(0)
	ftmp2 = float32(0)
	for i = 0; i < subl; i++ {
		ftmp1 += float32(*target.at(i) * *regressor.at(i))
		ftmp2 += float32(*regressor.at(i) * *regressor.at(i))
	}
	if float64(ftmp1) > float64(0) {
		return float32(ftmp1*ftmp1) / ftmp2
	} else {
		return float32(0)
	}
}

func enhancerInterface(out span[float32], input span[float32], decoder *decoderState) int {
	var enhBuf span[float32]
	var enhPeriod span[float32]
	var iblock int
	var isample int
	var lag int
	lag = 0
	var ilag int
	var i int
	var ioffset int
	var lastLag int
	var cc float32
	var maxcc float32
	var ftmp1 float32
	var ftmp2 float32
	var inPtr span[float32]
	var enhBufPtr1 span[float32]
	var enhBufPtr2 span[float32]
	var plcPred [80]float32
	var lpState [6]float32
	var downsampled [180]float32
	var inLen int
	inLen = 360
	var start int
	var plcBlockl int
	var inlag int
	enhBuf = span[float32]{data: decoder.enhBuf[:]}
	enhPeriod = span[float32]{data: decoder.enhPeriod[:]}
	copy(enhBuf.slice(640-decoder.frameSamples), enhBuf.add(decoder.frameSamples).slice(640-decoder.frameSamples))
	copy(enhBuf.add(640-decoder.frameSamples).slice(decoder.frameSamples), input.slice(decoder.frameSamples))
	if decoder.mode == 30 {
		plcBlockl = 80
	} else {
		plcBlockl = 40
	}
	ioffset = 0
	if decoder.mode == 20 {
		ioffset = 1
	}
	i = 3 - ioffset
	copy(enhPeriod.slice(8-i), enhPeriod.add(i).slice(8-i))
	copy(lpState[0:6], enhBuf.add((5+ioffset)*80+-126).slice(6))
	downSample(enhBuf.add((5+ioffset)*80+-120), span[float32]{data: lpFiltCoefsTbl[:]}, inLen-ioffset*80, span[float32]{data: lpState[:]}, span[float32]{data: downsampled[:]})
	for iblock = 0; iblock < 3-ioffset; iblock++ {
		lag = 10
		maxcc = xCorrCoef(span[float32]{data: downsampled[:], off: 60 + iblock*40}, span[float32]{data: downsampled[:], off: 60 + iblock*40 + -lag}, 40)
		for ilag = 11; ilag < 60; ilag++ {
			cc = xCorrCoef(span[float32]{data: downsampled[:], off: 60 + iblock*40}, span[float32]{data: downsampled[:], off: 60 + iblock*40 + -ilag}, 40)
			if cc > maxcc {
				maxcc = cc
				lag = ilag
			}
		}
		*enhPeriod.at(iblock + 5 + ioffset) = float32(float32(lag) * float32(2))
	}
	lastLag = lag * 2
	if decoder.prevEnhPl == 1 {
		inlag = int(*enhPeriod.at(5 + ioffset))
		lag = inlag - 1
		maxcc = xCorrCoef(input, input.add(lag), plcBlockl)
		for ilag = inlag; ilag <= inlag+1; ilag++ {
			cc = xCorrCoef(input, input.add(ilag), plcBlockl)
			if cc > maxcc {
				maxcc = cc
				lag = ilag
			}
		}
		*enhPeriod.at(5 + ioffset - 1) = float32(lag)
		inPtr = input.add(lag - 1)
		enhBufPtr1 = span[float32]{data: plcPred[:], off: plcBlockl - 1}
		if lag > plcBlockl {
			start = plcBlockl
		} else {
			start = lag
		}
		for isample = start; isample > 0; isample-- {
			*enhBufPtr1.at(0) = *inPtr.at(0)
			enhBufPtr1 = enhBufPtr1.add(-1)
			inPtr = inPtr.add(-1)
		}
		enhBufPtr2 = enhBuf.add(639 - decoder.frameSamples)
		for isample = plcBlockl - 1 - lag; isample >= 0; isample-- {
			*enhBufPtr1.at(0) = *enhBufPtr2.at(0)
			enhBufPtr1 = enhBufPtr1.add(-1)
			enhBufPtr2 = enhBufPtr2.add(-1)
		}
		ftmp2 = float32(0)
		ftmp1 = float32(0)
		for i = 0; i < plcBlockl; i++ {
			ftmp2 += float32(*enhBuf.at(639 - decoder.frameSamples - i) * *enhBuf.at(639 - decoder.frameSamples - i))
			ftmp1 += float32(plcPred[i] * plcPred[i])
		}
		ftmp1 = float32(math.Sqrt(float64(ftmp1 / float32(plcBlockl))))
		ftmp2 = float32(math.Sqrt(float64(ftmp2 / float32(plcBlockl))))
		if ftmp1 > float32(float32(2)*ftmp2) && float64(ftmp1) > float64(0) {
			for i = 0; i < plcBlockl-10; i++ {
				plcPred[i] *= float32(float32(2)*ftmp2) / ftmp1
			}
			for i = plcBlockl - 10; i < plcBlockl; i++ {
				plcPred[i] *= float32(float32(i-plcBlockl+10)*(float32(1)-float32(float32(2)*ftmp2)/ftmp1))/float32(10) + float32(float32(2)*ftmp2)/ftmp1
			}
		}
		enhBufPtr1 = enhBuf.add(639 - decoder.frameSamples)
		for i = 0; i < plcBlockl; i++ {
			ftmp1 = float32(i+1) / float32(plcBlockl+1)
			*enhBufPtr1.at(0) *= ftmp1
			*enhBufPtr1.at(0) += float32((float32(1) - ftmp1) * plcPred[plcBlockl-1-i])
			enhBufPtr1 = enhBufPtr1.add(-1)
		}
	}
	if decoder.mode == 20 {
		for iblock = 0; iblock < 2; iblock++ {
			enhancer(out.add(iblock*80), enhBuf, 640, (5+iblock)*80+40, float32(0.05), enhPeriod, span[float32]{data: enhPlocsTbl[:]}, 8)
		}
	} else {
		if decoder.mode == 30 {
			for iblock = 0; iblock < 3; iblock++ {
				enhancer(out.add(iblock*80), enhBuf, 640, (4+iblock)*80, float32(0.05), enhPeriod, span[float32]{data: enhPlocsTbl[:]}, 8)
			}
		}
	}
	return lastLag
}
