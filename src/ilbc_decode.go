// Port of RFC 3951 Appendix A: iLBC_decode.c.
// Copyright (C) The Internet Society (2004). See LICENSE.

package ilbc

func initDecode(decoder *decoderState, mode int, enhancement int) int16 {
	var i int
	decoder.mode = mode
	if mode == 30 {
		decoder.frameSamples = 240
		decoder.subframes = 6
		decoder.adaptiveSubframes = 4
		decoder.lpcAnalyses = 2
		decoder.frameBytes = 50
		decoder.frameWords = 25
		decoder.shortStateSamples = 58
		decoder.ulp = &ulp30Table
	} else {
		if mode == 20 {
			decoder.frameSamples = 160
			decoder.subframes = 4
			decoder.adaptiveSubframes = 2
			decoder.lpcAnalyses = 1
			decoder.frameBytes = 38
			decoder.frameWords = 19
			decoder.shortStateSamples = 57
			decoder.ulp = &ulp20Table
		} else {
			panic("ilbc: invalid internal mode")
		}
	}
	clear(decoder.syntMem[0:10])
	copy(decoder.lsfdeqold[0:10], lsfmeanTbl[0:10])
	clear(decoder.oldSyntdenum[0:66])
	for i = 0; i < 6; i++ {
		decoder.oldSyntdenum[i*11] = float32(1)
	}
	decoder.lastLag = 20
	decoder.prevLag = 120
	decoder.per = float32(0)
	decoder.consPLICount = 0
	decoder.prevPLI = 0
	decoder.prevLpc[0] = float32(1)
	clear(decoder.prevLpc[1:11])
	clear(decoder.prevResidual[0:240])
	decoder.seed = 777
	clear(decoder.hpomem[0:4])
	decoder.enhancement = enhancement
	clear(decoder.enhBuf[0:640])
	for i = 0; i < 8; i++ {
		decoder.enhPeriod[i] = float32(40)
	}
	decoder.prevEnhPl = 0
	return int16(decoder.frameSamples)
}

func decode(decoder *decoderState, decresidual span[float32], start int, idxForMax int, idxVec span[int], syntdenum span[float32], cbIndex span[int], gainIndex span[int], extraCbIndex span[int], extraGainIndex span[int], stateFirst int) {
	var reverseDecresidual [240]float32
	var mem [147]float32
	var k int
	var memlGotten int
	var nfor int
	var nback int
	var i int
	var diff int
	var startPos int
	var subcount int
	var subframe int
	diff = 80 - decoder.shortStateSamples
	if stateFirst == 1 {
		startPos = (start - 1) * 40
	} else {
		startPos = (start-1)*40 + diff
	}
	stateConstructW(idxForMax, idxVec, syntdenum.add((start-1)*11), decresidual.add(startPos), decoder.shortStateSamples)
	if stateFirst != 0 {
		clear(mem[0 : 147-decoder.shortStateSamples])
		copy(mem[147+-decoder.shortStateSamples:147+-decoder.shortStateSamples+decoder.shortStateSamples], decresidual.add(startPos).slice(decoder.shortStateSamples))
		constructCodebook(decresidual.add(startPos+decoder.shortStateSamples), extraCbIndex, extraGainIndex, span[float32]{data: mem[:], off: 147 + -stMemLTbl}, stMemLTbl, diff, 3)
	} else {
		for k = 0; k < diff; k++ {
			reverseDecresidual[k] = *decresidual.at((start+1)*40 - 1 - (k + decoder.shortStateSamples))
		}
		memlGotten = decoder.shortStateSamples
		for k = 0; k < memlGotten; k++ {
			mem[146-k] = *decresidual.at(startPos + k)
		}
		clear(mem[0 : 147-k])
		constructCodebook(span[float32]{data: reverseDecresidual[:]}, extraCbIndex, extraGainIndex, span[float32]{data: mem[:], off: 147 + -stMemLTbl}, stMemLTbl, diff, 3)
		for k = 0; k < diff; k++ {
			*decresidual.at(startPos - 1 - k) = reverseDecresidual[k]
		}
	}
	subcount = 0
	nfor = decoder.subframes - start - 1
	if nfor > 0 {
		clear(mem[0:67])
		copy(mem[67:147], decresidual.add((start-1)*40).slice(80))
		for subframe = 0; subframe < nfor; subframe++ {
			constructCodebook(decresidual.add((start+1+subframe)*40), cbIndex.add(subcount*3), gainIndex.add(subcount*3), span[float32]{data: mem[:], off: 147 + -memLfTbl[subcount]}, memLfTbl[subcount], 40, 3)
			copy(mem[0:107], mem[40:147])
			copy(mem[107:147], decresidual.add((start+1+subframe)*40).slice(40))
			subcount++
		}
	}
	nback = start - 1
	if nback > 0 {
		memlGotten = 40 * (decoder.subframes + 1 - start)
		if memlGotten > 147 {
			memlGotten = 147
		}
		for k = 0; k < memlGotten; k++ {
			mem[146-k] = *decresidual.at((start-1)*40 + k)
		}
		clear(mem[0 : 147-k])
		for subframe = 0; subframe < nback; subframe++ {
			constructCodebook(span[float32]{data: reverseDecresidual[:], off: subframe * 40}, cbIndex.add(subcount*3), gainIndex.add(subcount*3), span[float32]{data: mem[:], off: 147 + -memLfTbl[subcount]}, memLfTbl[subcount], 40, 3)
			copy(mem[0:107], mem[40:147])
			copy(mem[107:147], reverseDecresidual[subframe*40:subframe*40+40])
			subcount++
		}
		for i = 0; i < 40*nback; i++ {
			*decresidual.at(40*nback - i - 1) = reverseDecresidual[i]
		}
	}
}

func decodeFrame(decblock span[float32], bytes span[byte], decoder *decoderState, mode int) {
	var data [240]float32
	var lsfdeq [20]float32
	var pLCresidual [240]float32
	var pLClpc [11]float32
	var zeros [240]float32
	var one [11]float32
	var k int
	var i int
	var start [1]int
	var idxForMax [1]int
	var pos [1]int
	var lastpart [1]int
	var ulp int
	var lag int
	var ilag int
	var corrLen int
	var corrStart int
	var cc float32
	var maxcc float32
	var idxVec [80]int
	var gainIndex [12]int
	var extraGainIndex [3]int
	var cbIndex [12]int
	var extraCbIndex [3]int
	var lsfI [6]int
	var stateFirst [1]int
	var lastBit [1]int
	var pbytes [1]span[byte]
	var weightdenum [66]float32
	var orderPlusOne int
	var syntdenum [66]float32
	var decresidual [240]float32
	if mode > 0 {
		pbytes[0] = bytes
		pos[0] = 0
		for k = 0; k < 6; k++ {
			lsfI[k] = 0
		}
		start[0] = 0
		stateFirst[0] = 0
		idxForMax[0] = 0
		for k = 0; k < decoder.shortStateSamples; k++ {
			idxVec[k] = 0
		}
		for k = 0; k < 3; k++ {
			extraCbIndex[k] = 0
		}
		for k = 0; k < 3; k++ {
			extraGainIndex[k] = 0
		}
		for i = 0; i < decoder.adaptiveSubframes; i++ {
			for k = 0; k < 3; k++ {
				cbIndex[i*3+k] = 0
			}
		}
		for i = 0; i < decoder.adaptiveSubframes; i++ {
			for k = 0; k < 3; k++ {
				gainIndex[i*3+k] = 0
			}
		}
		for ulp = 0; ulp < 3; ulp++ {
			for k = 0; k < 3*decoder.lpcAnalyses; k++ {
				unpack(span[span[byte]]{data: pbytes[:]}, span[int]{data: lastpart[:]}, decoder.ulp.lsfBits[k][ulp], span[int]{data: pos[:]})
				packcombine(span[int]{data: lsfI[:], off: k}, lastpart[0], decoder.ulp.lsfBits[k][ulp])
			}
			unpack(span[span[byte]]{data: pbytes[:]}, span[int]{data: lastpart[:]}, decoder.ulp.startBits[ulp], span[int]{data: pos[:]})
			packcombine(span[int]{data: start[:]}, lastpart[0], decoder.ulp.startBits[ulp])
			unpack(span[span[byte]]{data: pbytes[:]}, span[int]{data: lastpart[:]}, decoder.ulp.startfirstBits[ulp], span[int]{data: pos[:]})
			packcombine(span[int]{data: stateFirst[:]}, lastpart[0], decoder.ulp.startfirstBits[ulp])
			unpack(span[span[byte]]{data: pbytes[:]}, span[int]{data: lastpart[:]}, decoder.ulp.scaleBits[ulp], span[int]{data: pos[:]})
			packcombine(span[int]{data: idxForMax[:]}, lastpart[0], decoder.ulp.scaleBits[ulp])
			for k = 0; k < decoder.shortStateSamples; k++ {
				unpack(span[span[byte]]{data: pbytes[:]}, span[int]{data: lastpart[:]}, decoder.ulp.stateBits[ulp], span[int]{data: pos[:]})
				packcombine(span[int]{data: idxVec[:], off: k}, lastpart[0], decoder.ulp.stateBits[ulp])
			}
			for k = 0; k < 3; k++ {
				unpack(span[span[byte]]{data: pbytes[:]}, span[int]{data: lastpart[:]}, decoder.ulp.extraCbIndex[k][ulp], span[int]{data: pos[:]})
				packcombine(span[int]{data: extraCbIndex[:], off: k}, lastpart[0], decoder.ulp.extraCbIndex[k][ulp])
			}
			for k = 0; k < 3; k++ {
				unpack(span[span[byte]]{data: pbytes[:]}, span[int]{data: lastpart[:]}, decoder.ulp.extraCbGain[k][ulp], span[int]{data: pos[:]})
				packcombine(span[int]{data: extraGainIndex[:], off: k}, lastpart[0], decoder.ulp.extraCbGain[k][ulp])
			}
			for i = 0; i < decoder.adaptiveSubframes; i++ {
				for k = 0; k < 3; k++ {
					unpack(span[span[byte]]{data: pbytes[:]}, span[int]{data: lastpart[:]}, decoder.ulp.cbIndex[i][k][ulp], span[int]{data: pos[:]})
					packcombine(span[int]{data: cbIndex[:], off: i*3 + k}, lastpart[0], decoder.ulp.cbIndex[i][k][ulp])
				}
			}
			for i = 0; i < decoder.adaptiveSubframes; i++ {
				for k = 0; k < 3; k++ {
					unpack(span[span[byte]]{data: pbytes[:]}, span[int]{data: lastpart[:]}, decoder.ulp.cbGain[i][k][ulp], span[int]{data: pos[:]})
					packcombine(span[int]{data: gainIndex[:], off: i*3 + k}, lastpart[0], decoder.ulp.cbGain[i][k][ulp])
				}
			}
		}
		unpack(span[span[byte]]{data: pbytes[:]}, span[int]{data: lastBit[:]}, 1, span[int]{data: pos[:]})
		if start[0] < 1 {
			mode = 0
		}
		// A 20 ms start index occupies two bits and cannot exceed 3.

		if decoder.mode == 30 && start[0] > 5 {
			mode = 0
		}
		if lastBit[0] == 1 {
			mode = 0
		}
		if mode == 1 {
			convertDecoderIndices(span[int]{data: cbIndex[:]})
			dequantizeLSF(span[float32]{data: lsfdeq[:]}, span[int]{data: lsfI[:]}, decoder.lpcAnalyses)
			checkLSF(span[float32]{data: lsfdeq[:]}, 10, decoder.lpcAnalyses)
			decoderInterpolateLSF(span[float32]{data: syntdenum[:]}, span[float32]{data: weightdenum[:]}, span[float32]{data: lsfdeq[:]}, 10, decoder)
			decode(decoder, span[float32]{data: decresidual[:]}, start[0], idxForMax[0], span[int]{data: idxVec[:]}, span[float32]{data: syntdenum[:]}, span[int]{data: cbIndex[:]}, span[int]{data: gainIndex[:]}, span[int]{data: extraCbIndex[:]}, span[int]{data: extraGainIndex[:]}, stateFirst[0])
			doThePLC(span[float32]{data: pLCresidual[:]}, span[float32]{data: pLClpc[:]}, 0, span[float32]{data: decresidual[:]}, span[float32]{data: syntdenum[:], off: 11 * (decoder.subframes - 1)}, decoder.lastLag, decoder)
			copy(decresidual[0:decoder.frameSamples], pLCresidual[0:decoder.frameSamples])
		}
	}
	if mode == 0 {
		clear(zeros[0:240])
		one[0] = float32(1)
		clear(one[1:11])
		start[0] = 0
		doThePLC(span[float32]{data: pLCresidual[:]}, span[float32]{data: pLClpc[:]}, 1, span[float32]{data: zeros[:]}, span[float32]{data: one[:]}, decoder.lastLag, decoder)
		copy(decresidual[0:decoder.frameSamples], pLCresidual[0:decoder.frameSamples])
		orderPlusOne = 11
		for i = 0; i < decoder.subframes; i++ {
			copy(syntdenum[i*orderPlusOne:i*orderPlusOne+orderPlusOne], pLClpc[0:orderPlusOne])
		}
	}
	if decoder.enhancement == 1 {
		decoder.lastLag = enhancerInterface(span[float32]{data: data[:]}, span[float32]{data: decresidual[:]}, decoder)
		if decoder.mode == 20 {
			i = 0
			syntFilter(span[float32]{data: data[:], off: i * 40}, span[float32]{data: decoder.oldSyntdenum[:], off: (i + decoder.subframes - 1) * 11}, 40, span[float32]{data: decoder.syntMem[:]})
			for i = 1; i < decoder.subframes; i++ {
				syntFilter(span[float32]{data: data[:], off: i * 40}, span[float32]{data: syntdenum[:], off: (i - 1) * 11}, 40, span[float32]{data: decoder.syntMem[:]})
			}
		} else {
			if decoder.mode == 30 {
				for i = 0; i < 2; i++ {
					syntFilter(span[float32]{data: data[:], off: i * 40}, span[float32]{data: decoder.oldSyntdenum[:], off: (i + decoder.subframes - 2) * 11}, 40, span[float32]{data: decoder.syntMem[:]})
				}
				for i = 2; i < decoder.subframes; i++ {
					syntFilter(span[float32]{data: data[:], off: i * 40}, span[float32]{data: syntdenum[:], off: (i - 2) * 11}, 40, span[float32]{data: decoder.syntMem[:]})
				}
			}
		}
	} else {
		corrLen = func() int {
			if decoder.frameSamples == 160 {
				return 40
			}
			return 80
		}()
		corrStart = decoder.frameSamples - corrLen
		lag = 20
		maxcc = xCorrCoef(span[float32]{data: decresidual[:], off: corrStart}, span[float32]{data: decresidual[:], off: corrStart - lag}, corrLen)
		for ilag = 21; ilag < 120; ilag++ {
			cc = xCorrCoef(span[float32]{data: decresidual[:], off: corrStart}, span[float32]{data: decresidual[:], off: corrStart - ilag}, corrLen)
			if cc > maxcc {
				maxcc = cc
				lag = ilag
			}
		}
		decoder.lastLag = lag
		copy(data[0:decoder.frameSamples], decresidual[0:decoder.frameSamples])
		for i = 0; i < decoder.subframes; i++ {
			syntFilter(span[float32]{data: data[:], off: i * 40}, span[float32]{data: syntdenum[:], off: i * 11}, 40, span[float32]{data: decoder.syntMem[:]})
		}
	}
	hpOutput(span[float32]{data: data[:]}, decoder.frameSamples, decblock, span[float32]{data: decoder.hpomem[:]})
	copy(decoder.oldSyntdenum[0:decoder.subframes*11], syntdenum[0:decoder.subframes*11])
	decoder.prevEnhPl = 0
	if mode == 0 {
		decoder.prevEnhPl = 1
	}
}
