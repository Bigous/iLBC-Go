// Port of RFC 3951 Appendix A: doCPLC.c.
// Copyright (C) The Internet Society (2004). See LICENSE.

package ilbc

import "math"

func compCorr(cc span[float32], gc span[float32], pm span[float32], buffer span[float32], lag int, bLen int, sRange int) {
	var i int
	var ftmp1 float32
	var ftmp2 float32
	var ftmp3 float32
	if bLen-sRange-lag < 0 {
		sRange = bLen - lag
	}
	ftmp1 = float32(0)
	ftmp2 = float32(0)
	ftmp3 = float32(0)
	for i = 0; i < sRange; i++ {
		ftmp1 += float32(*buffer.at(bLen - sRange + i) * *buffer.at(bLen - sRange + i - lag))
		ftmp2 += float32(*buffer.at(bLen - sRange + i - lag) * *buffer.at(bLen - sRange + i - lag))
		ftmp3 += float32(*buffer.at(bLen - sRange + i) * *buffer.at(bLen - sRange + i))
	}
	if float64(ftmp2) > float64(0) {
		*cc.at(0) = float32(ftmp1*ftmp1) / ftmp2
		*gc.at(0) = float32(math.Abs(float64(ftmp1 / ftmp2)))
		*pm.at(0) = float32(math.Abs(float64(ftmp1))) / float32(float32(math.Sqrt(float64(ftmp2)))*float32(math.Sqrt(float64(ftmp3))))
	} else {
		*cc.at(0) = float32(0)
		*gc.at(0) = float32(0)
		*pm.at(0) = float32(0)
	}
}

func doThePLC(pLCresidual span[float32], pLClpc span[float32], pLI int, decresidual span[float32], lpc span[float32], inlag int, decoder *decoderState) {
	var lag int
	lag = 20
	var randlag int
	var gain [1]float32
	var maxcc [1]float32
	var useGain float32
	var gainComp [1]float32
	var maxccComp [1]float32
	var per [1]float32
	var maxPer [1]float32
	var i int
	var pick int
	var useLag int
	var ftmp float32
	var randvec [240]float32
	var pitchfact float32
	var energy float32
	if pLI == 1 {
		if decoder.consPLICount < 9 {
			decoder.consPLICount += 1
		}
		if decoder.prevPLI != 1 {
			lag = inlag - 3
			compCorr(span[float32]{data: maxcc[:]}, span[float32]{data: gain[:]}, span[float32]{data: maxPer[:]}, span[float32]{data: decoder.prevResidual[:]}, lag, decoder.frameSamples, 60)
			for i = inlag - 2; i <= inlag+3; i++ {
				compCorr(span[float32]{data: maxccComp[:]}, span[float32]{data: gainComp[:]}, span[float32]{data: per[:]}, span[float32]{data: decoder.prevResidual[:]}, i, decoder.frameSamples, 60)
				if maxccComp[0] > maxcc[0] {
					maxcc[0] = maxccComp[0]
					gain[0] = gainComp[0]
					lag = i
					maxPer[0] = per[0]
				}
			}
		} else {
			lag = decoder.prevLag
			maxPer[0] = decoder.per
		}
		useGain = float32(1)
		if decoder.consPLICount*decoder.frameSamples > 1280 {
			useGain = float32(0)
		} else {
			if decoder.consPLICount*decoder.frameSamples > 960 {
				useGain = float32(0.5)
			} else {
				if decoder.consPLICount*decoder.frameSamples > 640 {
					useGain = float32(0.7)
				} else {
					if decoder.consPLICount*decoder.frameSamples > 320 {
						useGain = float32(0.9)
					}
				}
			}
		}
		ftmp = float32(math.Sqrt(float64(maxPer[0])))
		if ftmp > float32(0.7) {
			pitchfact = float32(1)
		} else {
			if ftmp > float32(0.4) {
				pitchfact = (ftmp - float32(0.4)) / float32(0.29999998)
			} else {
				pitchfact = float32(0)
			}
		}
		useLag = lag
		if lag < 80 {
			useLag = 2 * lag
		}
		energy = float32(0)
		for i = 0; i < decoder.frameSamples; i++ {
			decoder.seed = (decoder.seed*69069 + 1) & 2147483647
			randlag = 50 + int(int32(decoder.seed))%70
			pick = i - randlag
			if pick < 0 {
				randvec[i] = decoder.prevResidual[decoder.frameSamples+pick]
			} else {
				randvec[i] = randvec[pick]
			}
			pick = i - useLag
			if pick < 0 {
				*pLCresidual.at(i) = decoder.prevResidual[decoder.frameSamples+pick]
			} else {
				*pLCresidual.at(i) = *pLCresidual.at(pick)
			}
			if i < 80 {
				*pLCresidual.at(i) = float32(useGain * (float32(pitchfact**pLCresidual.at(i)) + float32((float32(1)-pitchfact)*randvec[i])))
			} else {
				if i < 160 {
					*pLCresidual.at(i) = float32(float32(float32(0.95)*useGain) * (float32(pitchfact**pLCresidual.at(i)) + float32((float32(1)-pitchfact)*randvec[i])))
				} else {
					*pLCresidual.at(i) = float32(float32(float32(0.9)*useGain) * (float32(pitchfact**pLCresidual.at(i)) + float32((float32(1)-pitchfact)*randvec[i])))
				}
			}
			energy += float32(*pLCresidual.at(i) * *pLCresidual.at(i))
		}
		if math.Sqrt(float64(energy/float32(decoder.frameSamples))) < float64(30) {
			gain[0] = float32(0)
			for i = 0; i < decoder.frameSamples; i++ {
				*pLCresidual.at(i) = float32(useGain * randvec[i])
			}
		}
		copy(pLClpc.slice(11), decoder.prevLpc[0:11])
	} else {
		copy(pLCresidual.slice(decoder.frameSamples), decresidual.slice(decoder.frameSamples))
		copy(pLClpc.slice(11), lpc.slice(11))
		decoder.consPLICount = 0
	}
	if pLI != 0 {
		decoder.prevLag = lag
		decoder.per = maxPer[0]
	}
	decoder.prevPLI = pLI
	copy(decoder.prevLpc[0:11], pLClpc.slice(11))
	copy(decoder.prevResidual[0:decoder.frameSamples], pLCresidual.slice(decoder.frameSamples))
}
