// Port of RFC 3951 Appendix A: iLBC_encode.c.
// Copyright (C) The Internet Society (2004). See LICENSE.

package ilbc

func initEncode(encoder *encoderState, mode int) int16 {
	encoder.mode = mode
	if mode == 30 {
		encoder.frameSamples = 240
		encoder.subframes = 6
		encoder.adaptiveSubframes = 4
		encoder.lpcAnalyses = 2
		encoder.frameBytes = 50
		encoder.frameWords = 25
		encoder.shortStateSamples = 58
		encoder.ulp = &ulp30Table
	} else {
		if mode == 20 {
			encoder.frameSamples = 160
			encoder.subframes = 4
			encoder.adaptiveSubframes = 2
			encoder.lpcAnalyses = 1
			encoder.frameBytes = 38
			encoder.frameWords = 19
			encoder.shortStateSamples = 57
			encoder.ulp = &ulp20Table
		} else {
			panic("ilbc: invalid internal mode")
		}
	}
	clear(encoder.anaMem[0:10])
	copy(encoder.lsfold[0:10], lsfmeanTbl[0:10])
	copy(encoder.lsfdeqold[0:10], lsfmeanTbl[0:10])
	clear(encoder.lpcBuffer[0:300])
	clear(encoder.hpimem[0:4])
	return int16(encoder.frameBytes)
}

func encodeFrame(bytes span[byte], block span[float32], encoder *encoderState) {
	var data [240]float32
	var residual [240]float32
	var reverseResidual [240]float32
	var start [1]int
	var idxForMax [1]int
	var idxVec [80]int
	var reverseDecresidual [240]float32
	var mem [147]float32
	var n int
	var k int
	var memlGotten int
	var nfor int
	var nback int
	var i int
	var pos [1]int
	var gainIndex [12]int
	var extraGainIndex [3]int
	var cbIndex [12]int
	var extraCbIndex [3]int
	var lsfI [6]int
	var pbytes [1]span[byte]
	var diff int
	var startPos int
	var stateFirst [1]int
	var en1 float32
	var en2 float32
	var index int
	var ulp int
	var firstpart [1]int
	var subcount int
	var subframe int
	var weightState [10]float32
	var syntdenum [66]float32
	var weightdenum [66]float32
	var decresidual [240]float32
	hpInput(block, encoder.frameSamples, span[float32]{data: data[:]}, span[float32]{data: encoder.hpimem[:]})
	encodeLPC(span[float32]{data: syntdenum[:]}, span[float32]{data: weightdenum[:]}, span[int]{data: lsfI[:]}, span[float32]{data: data[:]}, encoder)
	for n = 0; n < encoder.subframes; n++ {
		anaFilter(span[float32]{data: data[:], off: n * 40}, span[float32]{data: syntdenum[:], off: n * 11}, 40, span[float32]{data: residual[:], off: n * 40}, span[float32]{data: encoder.anaMem[:]})
	}
	start[0] = frameClassify(encoder, span[float32]{data: residual[:]})
	diff = 80 - encoder.shortStateSamples
	en1 = float32(0)
	index = (start[0] - 1) * 40
	for i = 0; i < encoder.shortStateSamples; i++ {
		en1 += float32(residual[index+i] * residual[index+i])
	}
	en2 = float32(0)
	index = (start[0]-1)*40 + diff
	for i = 0; i < encoder.shortStateSamples; i++ {
		en2 += float32(residual[index+i] * residual[index+i])
	}
	if en1 > en2 {
		stateFirst[0] = 1
		startPos = (start[0] - 1) * 40
	} else {
		stateFirst[0] = 0
		startPos = (start[0]-1)*40 + diff
	}
	stateSearchW(encoder, span[float32]{data: residual[:], off: startPos}, span[float32]{data: syntdenum[:], off: (start[0] - 1) * 11}, span[float32]{data: weightdenum[:], off: (start[0] - 1) * 11}, span[int]{data: idxForMax[:]}, span[int]{data: idxVec[:]}, encoder.shortStateSamples, stateFirst[0])
	stateConstructW(idxForMax[0], span[int]{data: idxVec[:]}, span[float32]{data: syntdenum[:], off: (start[0] - 1) * 11}, span[float32]{data: decresidual[:], off: startPos}, encoder.shortStateSamples)
	if stateFirst[0] != 0 {
		clear(mem[0 : 147-encoder.shortStateSamples])
		copy(mem[147+-encoder.shortStateSamples:147+-encoder.shortStateSamples+encoder.shortStateSamples], decresidual[startPos:startPos+encoder.shortStateSamples])
		clear(weightState[0:10])
		searchCodebook(encoder, span[int]{data: extraCbIndex[:]}, span[int]{data: extraGainIndex[:]}, span[float32]{data: residual[:], off: startPos + encoder.shortStateSamples}, span[float32]{data: mem[:], off: 147 + -stMemLTbl}, stMemLTbl, diff, 3, span[float32]{data: weightdenum[:], off: start[0] * 11}, span[float32]{data: weightState[:]}, 0)
		constructCodebook(span[float32]{data: decresidual[:], off: startPos + encoder.shortStateSamples}, span[int]{data: extraCbIndex[:]}, span[int]{data: extraGainIndex[:]}, span[float32]{data: mem[:], off: 147 + -stMemLTbl}, stMemLTbl, diff, 3)
	} else {
		for k = 0; k < diff; k++ {
			reverseResidual[k] = residual[(start[0]+1)*40-1-(k+encoder.shortStateSamples)]
		}
		memlGotten = encoder.shortStateSamples
		for k = 0; k < memlGotten; k++ {
			mem[146-k] = decresidual[startPos+k]
		}
		clear(mem[0 : 147-k])
		clear(weightState[0:10])
		searchCodebook(encoder, span[int]{data: extraCbIndex[:]}, span[int]{data: extraGainIndex[:]}, span[float32]{data: reverseResidual[:]}, span[float32]{data: mem[:], off: 147 + -stMemLTbl}, stMemLTbl, diff, 3, span[float32]{data: weightdenum[:], off: (start[0] - 1) * 11}, span[float32]{data: weightState[:]}, 0)
		constructCodebook(span[float32]{data: reverseDecresidual[:]}, span[int]{data: extraCbIndex[:]}, span[int]{data: extraGainIndex[:]}, span[float32]{data: mem[:], off: 147 + -stMemLTbl}, stMemLTbl, diff, 3)
		for k = 0; k < diff; k++ {
			decresidual[startPos-1-k] = reverseDecresidual[k]
		}
	}
	subcount = 0
	nfor = encoder.subframes - start[0] - 1
	if nfor > 0 {
		clear(mem[0:67])
		copy(mem[67:147], decresidual[(start[0]-1)*40:(start[0]-1)*40+80])
		clear(weightState[0:10])
		for subframe = 0; subframe < nfor; subframe++ {
			searchCodebook(encoder, span[int]{data: cbIndex[:], off: subcount * 3}, span[int]{data: gainIndex[:], off: subcount * 3}, span[float32]{data: residual[:], off: (start[0] + 1 + subframe) * 40}, span[float32]{data: mem[:], off: 147 + -memLfTbl[subcount]}, memLfTbl[subcount], 40, 3, span[float32]{data: weightdenum[:], off: (start[0] + 1 + subframe) * 11}, span[float32]{data: weightState[:]}, subcount+1)
			constructCodebook(span[float32]{data: decresidual[:], off: (start[0] + 1 + subframe) * 40}, span[int]{data: cbIndex[:], off: subcount * 3}, span[int]{data: gainIndex[:], off: subcount * 3}, span[float32]{data: mem[:], off: 147 + -memLfTbl[subcount]}, memLfTbl[subcount], 40, 3)
			copy(mem[0:107], mem[40:147])
			copy(mem[107:147], decresidual[(start[0]+1+subframe)*40:(start[0]+1+subframe)*40+40])
			clear(weightState[0:10])
			subcount++
		}
	}
	nback = start[0] - 1
	if nback > 0 {
		for n = 0; n < nback; n++ {
			for k = 0; k < 40; k++ {
				reverseResidual[n*40+k] = residual[(start[0]-1)*40-1-n*40-k]
				reverseDecresidual[n*40+k] = decresidual[(start[0]-1)*40-1-n*40-k]
			}
		}
		memlGotten = 40 * (encoder.subframes + 1 - start[0])
		if memlGotten > 147 {
			memlGotten = 147
		}
		for k = 0; k < memlGotten; k++ {
			mem[146-k] = decresidual[(start[0]-1)*40+k]
		}
		clear(mem[0 : 147-k])
		clear(weightState[0:10])
		for subframe = 0; subframe < nback; subframe++ {
			searchCodebook(encoder, span[int]{data: cbIndex[:], off: subcount * 3}, span[int]{data: gainIndex[:], off: subcount * 3}, span[float32]{data: reverseResidual[:], off: subframe * 40}, span[float32]{data: mem[:], off: 147 + -memLfTbl[subcount]}, memLfTbl[subcount], 40, 3, span[float32]{data: weightdenum[:], off: (start[0] - 2 - subframe) * 11}, span[float32]{data: weightState[:]}, subcount+1)
			constructCodebook(span[float32]{data: reverseDecresidual[:], off: subframe * 40}, span[int]{data: cbIndex[:], off: subcount * 3}, span[int]{data: gainIndex[:], off: subcount * 3}, span[float32]{data: mem[:], off: 147 + -memLfTbl[subcount]}, memLfTbl[subcount], 40, 3)
			copy(mem[0:107], mem[40:147])
			copy(mem[107:147], reverseDecresidual[subframe*40:subframe*40+40])
			clear(weightState[0:10])
			subcount++
		}
		for i = 0; i < 40*nback; i++ {
			decresidual[40*nback-i-1] = reverseDecresidual[i]
		}
	}
	convertEncoderIndices(span[int]{data: cbIndex[:]})
	pbytes[0] = bytes
	pos[0] = 0
	for ulp = 0; ulp < 3; ulp++ {
		for k = 0; k < 3*encoder.lpcAnalyses; k++ {
			packsplit(span[int]{data: lsfI[:], off: k}, span[int]{data: firstpart[:]}, span[int]{data: lsfI[:], off: k}, encoder.ulp.lsfBits[k][ulp], encoder.ulp.lsfBits[k][ulp]+encoder.ulp.lsfBits[k][ulp+1]+encoder.ulp.lsfBits[k][ulp+2])
			dopack(span[span[byte]]{data: pbytes[:]}, firstpart[0], encoder.ulp.lsfBits[k][ulp], span[int]{data: pos[:]})
		}
		packsplit(span[int]{data: start[:]}, span[int]{data: firstpart[:]}, span[int]{data: start[:]}, encoder.ulp.startBits[ulp], encoder.ulp.startBits[ulp]+encoder.ulp.startBits[ulp+1]+encoder.ulp.startBits[ulp+2])
		dopack(span[span[byte]]{data: pbytes[:]}, firstpart[0], encoder.ulp.startBits[ulp], span[int]{data: pos[:]})
		packsplit(span[int]{data: stateFirst[:]}, span[int]{data: firstpart[:]}, span[int]{data: stateFirst[:]}, encoder.ulp.startfirstBits[ulp], encoder.ulp.startfirstBits[ulp]+encoder.ulp.startfirstBits[ulp+1]+encoder.ulp.startfirstBits[ulp+2])
		dopack(span[span[byte]]{data: pbytes[:]}, firstpart[0], encoder.ulp.startfirstBits[ulp], span[int]{data: pos[:]})
		packsplit(span[int]{data: idxForMax[:]}, span[int]{data: firstpart[:]}, span[int]{data: idxForMax[:]}, encoder.ulp.scaleBits[ulp], encoder.ulp.scaleBits[ulp]+encoder.ulp.scaleBits[ulp+1]+encoder.ulp.scaleBits[ulp+2])
		dopack(span[span[byte]]{data: pbytes[:]}, firstpart[0], encoder.ulp.scaleBits[ulp], span[int]{data: pos[:]})
		for k = 0; k < encoder.shortStateSamples; k++ {
			packsplit(span[int]{data: idxVec[:], off: k}, span[int]{data: firstpart[:]}, span[int]{data: idxVec[:], off: k}, encoder.ulp.stateBits[ulp], encoder.ulp.stateBits[ulp]+encoder.ulp.stateBits[ulp+1]+encoder.ulp.stateBits[ulp+2])
			dopack(span[span[byte]]{data: pbytes[:]}, firstpart[0], encoder.ulp.stateBits[ulp], span[int]{data: pos[:]})
		}
		for k = 0; k < 3; k++ {
			packsplit(span[int]{data: extraCbIndex[:], off: k}, span[int]{data: firstpart[:]}, span[int]{data: extraCbIndex[:], off: k}, encoder.ulp.extraCbIndex[k][ulp], encoder.ulp.extraCbIndex[k][ulp]+encoder.ulp.extraCbIndex[k][ulp+1]+encoder.ulp.extraCbIndex[k][ulp+2])
			dopack(span[span[byte]]{data: pbytes[:]}, firstpart[0], encoder.ulp.extraCbIndex[k][ulp], span[int]{data: pos[:]})
		}
		for k = 0; k < 3; k++ {
			packsplit(span[int]{data: extraGainIndex[:], off: k}, span[int]{data: firstpart[:]}, span[int]{data: extraGainIndex[:], off: k}, encoder.ulp.extraCbGain[k][ulp], encoder.ulp.extraCbGain[k][ulp]+encoder.ulp.extraCbGain[k][ulp+1]+encoder.ulp.extraCbGain[k][ulp+2])
			dopack(span[span[byte]]{data: pbytes[:]}, firstpart[0], encoder.ulp.extraCbGain[k][ulp], span[int]{data: pos[:]})
		}
		for i = 0; i < encoder.adaptiveSubframes; i++ {
			for k = 0; k < 3; k++ {
				packsplit(span[int]{data: cbIndex[:], off: i*3 + k}, span[int]{data: firstpart[:]}, span[int]{data: cbIndex[:], off: i*3 + k}, encoder.ulp.cbIndex[i][k][ulp], encoder.ulp.cbIndex[i][k][ulp]+encoder.ulp.cbIndex[i][k][ulp+1]+encoder.ulp.cbIndex[i][k][ulp+2])
				dopack(span[span[byte]]{data: pbytes[:]}, firstpart[0], encoder.ulp.cbIndex[i][k][ulp], span[int]{data: pos[:]})
			}
		}
		for i = 0; i < encoder.adaptiveSubframes; i++ {
			for k = 0; k < 3; k++ {
				packsplit(span[int]{data: gainIndex[:], off: i*3 + k}, span[int]{data: firstpart[:]}, span[int]{data: gainIndex[:], off: i*3 + k}, encoder.ulp.cbGain[i][k][ulp], encoder.ulp.cbGain[i][k][ulp]+encoder.ulp.cbGain[i][k][ulp+1]+encoder.ulp.cbGain[i][k][ulp+2])
				dopack(span[span[byte]]{data: pbytes[:]}, firstpart[0], encoder.ulp.cbGain[i][k][ulp], span[int]{data: pos[:]})
			}
		}
	}
	dopack(span[span[byte]]{data: pbytes[:]}, 0, 1, span[int]{data: pos[:]})
}
