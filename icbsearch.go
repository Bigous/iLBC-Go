// Port of RFC 3951 Appendix A: iCBSearch.c.
// Copyright (C) The Internet Society (2004). See LICENSE.

package ilbc

import "math"

// codebookDotProduct checks the vector extent once, allowing the compiler to
// eliminate bounds checks inside the codebook search's innermost loop. Keep
// accumulation sequential and round each product to match the RFC reference.
func codebookDotProduct(left, right []float32) float32 {
	right = right[:len(left)]
	var sum float32
	for i, value := range left {
		sum += float32(value * right[i])
	}
	return sum
}

func searchCodebook(encoder *encoderState, index span[int], gainIndex span[int], intarget span[float32], mem span[float32], lMem int, lTarget int, nStages int, weightDenum span[float32], weightState span[float32], block int) {
	var i int
	var j int
	var icount int
	var stage int
	var bestIndex [1]int
	var valueRange int
	var counter int
	var maxMeasure [1]float32
	var gain [1]float32
	var measure float32
	var crossDot float32
	var ftmp float32
	var gains [3]float32
	var target [40]float32
	var baseIndex int
	var sInd int
	var eInd int
	var baseSize int
	var sIndAug int
	sIndAug = 0
	var eIndAug int
	eIndAug = 0
	var buf [207]float32
	var invenergy [256]float32
	var energy [256]float32
	var pp span[float32]
	var ppi span[float32]
	ppi = span[float32]{}
	var ppo span[float32]
	ppo = span[float32]{}
	var ppe span[float32]
	ppe = span[float32]{}
	var cbvectors [147]float32
	var tene float32
	var cene float32
	var cvec [40]float32
	var augVec [40]float32
	clear(cvec[0:40])
	baseSize = lMem - lTarget + 1
	if lTarget == 40 {
		baseSize = lMem - lTarget + 1 + lTarget/2
	}
	copy(buf[0:10], weightState.slice(10))
	copy(buf[10:10+lMem], mem.slice(lMem))
	copy(buf[10+lMem:10+lMem+lTarget], intarget.slice(lTarget))
	allPoleFilter(span[float32]{data: buf[:], off: 10}, weightDenum, lMem+lTarget, 10)
	copy(target[0:lTarget], buf[10+lMem:10+lMem+lTarget])
	tene = float32(0)
	for i = 0; i < lTarget; i++ {
		tene += float32(target[i] * target[i])
	}
	filteredCBvecs(span[float32]{data: cbvectors[:]}, span[float32]{data: buf[:], off: 10}, lMem)
	for stage = 0; stage < nStages; stage++ {
		valueRange = searchRangeTbl[block][stage]
		maxMeasure[0] = float32(-1e+07)
		gain[0] = float32(0)
		bestIndex[0] = 0
		pp = span[float32]{data: buf[:], off: 10 + lMem + -lTarget}
		crossDot = codebookDotProduct(target[:lTarget], pp.slice(lTarget))
		pp = pp.add(lTarget)
		if stage == 0 {
			ppe = span[float32]{data: energy[:]}
			ppi = span[float32]{data: buf[:], off: 10 + lMem + -lTarget + -1}
			ppo = span[float32]{data: buf[:], off: 10 + lMem + -1}
			*ppe.at(0) = float32(0)
			pp = span[float32]{data: buf[:], off: 10 + lMem + -lTarget}
			for j = 0; j < lTarget; j++ {
				*ppe.at(0) += float32(*pp.at(0) * *pp.at(0))
				pp = pp.add(1)
			}
			if float64(*ppe.at(0)) > float64(0) {
				invenergy[0] = float32(1) / (*ppe.at(0) + float32(2.220446e-16))
			} else {
				invenergy[0] = float32(0)
			}
			ppe = ppe.add(1)
			measure = float32(-1e+07)
			if float64(crossDot) > float64(0) {
				measure = float32(float32(crossDot*crossDot) * invenergy[0])
			}
		} else {
			measure = float32(float32(crossDot*crossDot) * invenergy[0])
		}
		ftmp = float32(crossDot * invenergy[0])
		if measure > maxMeasure[0] && math.Abs(float64(ftmp)) < float64(1.2999999523162842) {
			bestIndex[0] = 0
			maxMeasure[0] = measure
			gain[0] = ftmp
		}
		for icount = 1; icount < valueRange; icount++ {
			pp = span[float32]{data: buf[:], off: 10 + lMem + -lTarget + -icount}
			crossDot = codebookDotProduct(target[:lTarget], pp.slice(lTarget))
			pp = pp.add(lTarget)
			if stage == 0 {
				*ppe.at(0) = energy[icount-1] + float32(*ppi.at(0)**ppi.at(0)) - float32(*ppo.at(0)**ppo.at(0))
				ppe = ppe.add(1)
				ppo = ppo.add(-1)
				ppi = ppi.add(-1)
				if float64(energy[icount]) > float64(0) {
					invenergy[icount] = float32(1) / (energy[icount] + float32(2.220446e-16))
				} else {
					invenergy[icount] = float32(0)
				}
				measure = float32(-1e+07)
				if float64(crossDot) > float64(0) {
					measure = float32(float32(crossDot*crossDot) * invenergy[icount])
				}
			} else {
				measure = float32(float32(crossDot*crossDot) * invenergy[icount])
			}
			ftmp = float32(crossDot * invenergy[icount])
			if measure > maxMeasure[0] && math.Abs(float64(ftmp)) < float64(1.2999999523162842) {
				bestIndex[0] = icount
				maxMeasure[0] = measure
				gain[0] = ftmp
			}
		}
		if lTarget == 40 {
			searchAugmentedCB(20, 39, stage, baseSize-lTarget/2, span[float32]{data: target[:]}, span[float32]{data: buf[:], off: 10 + lMem}, span[float32]{data: maxMeasure[:]}, span[int]{data: bestIndex[:]}, span[float32]{data: gain[:]}, span[float32]{data: energy[:]}, span[float32]{data: invenergy[:]})
		}
		baseIndex = bestIndex[0]
		{ // RFC fixes CBRESRANGE at 34.

			sIndAug = 0
			eIndAug = 0
			sInd = baseIndex - 17
			eInd = sInd + 34
			if lTarget == 40 {
				if sInd < 0 {
					sIndAug = 40 + sInd
					eIndAug = 39
					sInd = 0
				} else {
					if baseIndex < baseSize-20 {
						if eInd > valueRange {
							sInd -= eInd - valueRange
							eInd = valueRange
						}
					} else {
						if sInd < baseSize-20 {
							sIndAug = 20
							sInd = 0
							eInd = 0
							eIndAug = 53
							if eIndAug > 39 {
								eInd = eIndAug - 39
								eIndAug = 39
							}
						} else {
							sIndAug = 20 + sInd - (baseSize - 20)
							eIndAug = 39
							sInd = 0
							eInd = 34 - (eIndAug - sIndAug + 1)
						}
					}
				}
			} else {
				if sInd < 0 {
					eInd -= sInd
					sInd = 0
				}
				if eInd > valueRange {
					sInd -= eInd - valueRange
					eInd = valueRange
				}
			}
		}
		counter = sInd
		sInd += baseSize
		eInd += baseSize
		if stage == 0 {
			ppe = span[float32]{data: energy[:], off: baseSize}
			*ppe.at(0) = float32(0)
			pp = span[float32]{data: cbvectors[:], off: lMem + -lTarget}
			for j = 0; j < lTarget; j++ {
				*ppe.at(0) += float32(*pp.at(0) * *pp.at(0))
				pp = pp.add(1)
			}
			ppi = span[float32]{data: cbvectors[:], off: lMem + -1 + -lTarget}
			ppo = span[float32]{data: cbvectors[:], off: lMem + -1}
			for j = 0; j < valueRange-1; j++ {
				*ppe.add(1).at(0) = *ppe.at(0) + float32(*ppi.at(0)**ppi.at(0)) - float32(*ppo.at(0)**ppo.at(0))
				ppo = ppo.add(-1)
				ppi = ppi.add(-1)
				ppe = ppe.add(1)
			}
		}
		for icount = sInd; icount < eInd; icount++ {
			pp = span[float32]{data: cbvectors[:], off: lMem + -counter + -lTarget}
			counter++
			crossDot = codebookDotProduct(target[:lTarget], pp.slice(lTarget))
			pp = pp.add(lTarget)
			if float64(energy[icount]) > float64(0) {
				invenergy[icount] = float32(1) / (energy[icount] + float32(2.220446e-16))
			} else {
				invenergy[icount] = float32(0)
			}
			if stage == 0 {
				measure = float32(-1e+07)
				if float64(crossDot) > float64(0) {
					measure = float32(float32(crossDot*crossDot) * invenergy[icount])
				}
			} else {
				measure = float32(float32(crossDot*crossDot) * invenergy[icount])
			}
			ftmp = float32(crossDot * invenergy[icount])
			if measure > maxMeasure[0] && math.Abs(float64(ftmp)) < float64(1.2999999523162842) {
				bestIndex[0] = icount
				maxMeasure[0] = measure
				gain[0] = ftmp
			}
		}
		if lTarget == 40 && sIndAug != 0 {
			searchAugmentedCB(sIndAug, eIndAug, stage, 2*baseSize-20, span[float32]{data: target[:]}, span[float32]{data: cbvectors[:], off: lMem}, span[float32]{data: maxMeasure[:]}, span[int]{data: bestIndex[:]}, span[float32]{data: gain[:]}, span[float32]{data: energy[:]}, span[float32]{data: invenergy[:]})
		}
		*index.at(stage) = bestIndex[0]
		if stage == 0 {
			gain[0] = min(max(gain[0], float32(0)), float32(1.3))
			gain[0] = gainquant(gain[0], float32(1), 32, gainIndex.add(stage))
		} else {
			if stage == 1 {
				gain[0] = gainquant(gain[0], float32(math.Abs(float64(gains[stage-1]))), 16, gainIndex.add(stage))
			} else {
				gain[0] = gainquant(gain[0], float32(math.Abs(float64(gains[stage-1]))), 8, gainIndex.add(stage))
			}
		}
		if lTarget == 80-encoder.shortStateSamples {
			if *index.at(stage) < baseSize {
				pp = span[float32]{data: buf[:], off: 10 + lMem + -lTarget + -*index.at(stage)}
			} else {
				pp = span[float32]{data: cbvectors[:], off: lMem + -lTarget + -*index.at(stage) + baseSize}
			}
		} else {
			if *index.at(stage) < baseSize {
				if *index.at(stage) < baseSize-20 {
					pp = span[float32]{data: buf[:], off: 10 + lMem + -lTarget + -*index.at(stage)}
				} else {
					createAugmentedVec(*index.at(stage)-baseSize+40, span[float32]{data: buf[:], off: 10 + lMem}, span[float32]{data: augVec[:]})
					pp = span[float32]{data: augVec[:]}
				}
			} else {
				var filterno int
				var position int
				filterno = *index.at(stage) / baseSize
				position = *index.at(stage) - filterno*baseSize
				if position < baseSize-20 {
					pp = span[float32]{data: cbvectors[:], off: filterno*lMem + -lTarget + -*index.at(stage) + filterno*baseSize}
				} else {
					createAugmentedVec(*index.at(stage)-(filterno+1)*baseSize+40, span[float32]{data: cbvectors[:], off: filterno * lMem}, span[float32]{data: augVec[:]})
					pp = span[float32]{data: augVec[:]}
				}
			}
		}
		for j = 0; j < lTarget; j++ {
			cvec[j] += float32(gain[0] * *pp.at(0))
			target[j] -= float32(gain[0] * *pp.at(0))
			pp = pp.add(1)
		}
		gains[stage] = gain[0]
	}
	cene = float32(0)
	for i = 0; i < lTarget; i++ {
		cene += float32(cvec[i] * cvec[i])
	}
	j = *gainIndex.at(0)
	for i = *gainIndex.at(0); i < 32; i++ {
		ftmp = float32(float32(cene*gainSq5Tbl[i]) * gainSq5Tbl[i])
		if ftmp < float32(float32(tene*gains[0])*gains[0]) && float64(gainSq5Tbl[j]) < float64(float64(2)*float64(gains[0])) {
			j = i
		}
	}
	*gainIndex.at(0) = j
}
