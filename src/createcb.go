// Port of RFC 3951 Appendix A: createCB.c.
// Copyright (C) The Internet Society (2004). See LICENSE.

package ilbc

import "math"

func filteredCBvecs(cbvectors span[float32], mem span[float32], lMem int) {
	var j int
	var k int
	var pp span[float32]
	var pp1 span[float32]
	var tempbuff2 [155]float32
	var pos span[float32]
	clear(tempbuff2[0:3])
	copy(tempbuff2[3:3+lMem], mem.slice(lMem))
	clear(tempbuff2[lMem+4-1 : lMem+4-1+5])
	pos = cbvectors
	clear(pos.slice(lMem))
	for k = 0; k < lMem; k++ {
		pp = span[float32]{data: tempbuff2[:], off: k}
		pp1 = span[float32]{data: cbfiltersTbl[:], off: 7}
		for j = 0; j < 8; j++ {
			*pos.at(0) += float32(*pp.at(0) * *pp1.at(0))
			pp = pp.add(1)
			pp1 = pp1.add(-1)
		}
		pos = pos.add(1)
	}
}

func searchAugmentedCB(low int, high int, stage int, startIndex int, target span[float32], buffer span[float32], maxMeasure span[float32], bestIndex span[int], gain span[float32], energy span[float32], invenergy span[float32]) {
	var icount int
	var ilow int
	var j int
	var tmpIndex int
	var pp span[float32]
	var ppo span[float32]
	var ppi span[float32]
	var ppe span[float32]
	var crossDot float32
	var alfa float32
	var weighted float32
	var measure float32
	var nrjRecursive float32
	var ftmp float32
	nrjRecursive = float32(0)
	pp = buffer.add(-low + 1)
	for j = 0; j < low-5; j++ {
		nrjRecursive += float32(*pp.at(0) * *pp.at(0))
		pp = pp.add(1)
	}
	ppe = buffer.add(-low)
	for icount = low; icount <= high; icount++ {
		tmpIndex = startIndex + icount - 20
		ilow = icount - 4
		nrjRecursive = nrjRecursive + float32(*ppe.at(0)**ppe.at(0))
		ppe = ppe.add(-1)
		*energy.at(tmpIndex) = nrjRecursive
		crossDot = float32(0)
		pp = buffer.add(-icount)
		for j = 0; j < ilow; j++ {
			crossDot += float32(*target.at(j) * *pp.at(0))
			pp = pp.add(1)
		}
		alfa = float32(0.2)
		ppo = buffer.add(-4)
		ppi = buffer.add(-icount + -4)
		for j = ilow; j < icount; j++ {
			weighted = float32((float32(1)-alfa)**ppo.at(0)) + float32(alfa**ppi.at(0))
			ppo = ppo.add(1)
			ppi = ppi.add(1)
			*energy.at(tmpIndex) += float32(weighted * weighted)
			crossDot += float32(*target.at(j) * weighted)
			alfa += float32(0.2)
		}
		pp = buffer.add(-icount)
		for j = icount; j < 40; j++ {
			*energy.at(tmpIndex) += float32(*pp.at(0) * *pp.at(0))
			crossDot += float32(*target.at(j) * *pp.at(0))
			pp = pp.add(1)
		}
		if float64(*energy.at(tmpIndex)) > float64(0) {
			*invenergy.at(tmpIndex) = float32(1) / (*energy.at(tmpIndex) + float32(2.220446e-16))
		} else {
			*invenergy.at(tmpIndex) = float32(0)
		}
		if stage == 0 {
			measure = float32(-1e+07)
			if float64(crossDot) > float64(0) {
				measure = float32(float32(crossDot*crossDot) * *invenergy.at(tmpIndex))
			}
		} else {
			measure = float32(float32(crossDot*crossDot) * *invenergy.at(tmpIndex))
		}
		ftmp = float32(crossDot * *invenergy.at(tmpIndex))
		if measure > *maxMeasure.at(0) && math.Abs(float64(ftmp)) < float64(1.2999999523162842) {
			*bestIndex.at(0) = tmpIndex
			*maxMeasure.at(0) = measure
			*gain.at(0) = ftmp
		}
	}
}

func createAugmentedVec(index int, buffer span[float32], cbVec span[float32]) {
	var ilow int
	var j int
	var pp span[float32]
	var ppo span[float32]
	var ppi span[float32]
	var alfa float32
	var alfa1 float32
	var weighted float32
	ilow = index - 5
	pp = buffer.add(-index)
	copy(cbVec.slice(4*index/4), pp.slice(4*index/4))
	alfa1 = float32(0.2)
	alfa = float32(0)
	ppo = buffer.add(-5)
	ppi = buffer.add(-index + -5)
	for j = ilow; j < index; j++ {
		weighted = float32((float32(1)-alfa)**ppo.at(0)) + float32(alfa**ppi.at(0))
		ppo = ppo.add(1)
		ppi = ppi.add(1)
		*cbVec.at(j) = weighted
		alfa += alfa1
	}
	pp = buffer.add(-index)
	copy(cbVec.add(index).slice(4*(40-index)/4), pp.slice(4*(40-index)/4))
}
