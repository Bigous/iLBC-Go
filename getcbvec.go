// Port of RFC 3951 Appendix A: getCBvec.c.
// Copyright (C) The Internet Society (2004). See LICENSE.

package ilbc

func getCBvec(cbvec span[float32], mem span[float32], index int, lMem int, cbveclen int) {
	var j int
	var k int
	var n int
	var memInd int
	var sFilt int
	var tmpbuf [147]float32
	var baseSize int
	var ilow int
	var ihigh int
	var alfa float32
	var alfa1 float32
	baseSize = lMem - cbveclen + 1
	if cbveclen == 40 {
		baseSize += cbveclen / 2
	}
	if index < lMem-cbveclen+1 {
		k = index + cbveclen
		copy(cbvec.slice(cbveclen), mem.add(lMem+-k).slice(cbveclen))
	} else {
		if index < baseSize {
			k = 2*(index-(lMem-cbveclen+1)) + cbveclen
			ihigh = k / 2
			ilow = ihigh - 5
			copy(cbvec.slice(ilow), mem.add(lMem+-(k/2)).slice(ilow))
			alfa1 = float32(0.2)
			alfa = float32(0)
			for j = ilow; j < ihigh; j++ {
				*cbvec.at(j) = float32((float32(1)-alfa)**mem.at(lMem - k/2 + j)) + float32(alfa**mem.at(lMem - k + j))
				alfa += alfa1
			}
			copy(cbvec.add(ihigh).slice(cbveclen-ihigh), mem.add(lMem+-k+ihigh).slice(cbveclen-ihigh))
		} else {
			if index-baseSize < lMem-cbveclen+1 {
				var tempbuff2 [156]float32
				var pos span[float32]
				var pp span[float32]
				var pp1 span[float32]
				clear(tempbuff2[0:4])
				copy(tempbuff2[4:4+lMem], mem.slice(lMem))
				clear(tempbuff2[lMem+4 : lMem+4+5])
				k = index - baseSize + cbveclen
				sFilt = lMem - k
				memInd = sFilt + 1 - 4
				pos = cbvec
				clear(pos.slice(cbveclen))
				for n = 0; n < cbveclen; n++ {
					pp = span[float32]{data: tempbuff2[:], off: memInd + n + 4}
					pp1 = span[float32]{data: cbfiltersTbl[:], off: 7}
					for j = 0; j < 8; j++ {
						*pos.at(0) += float32(*pp.at(0) * *pp1.at(0))
						pp = pp.add(1)
						pp1 = pp1.add(-1)
					}
					pos = pos.add(1)
				}
			} else {
				var tempbuff2 [156]float32
				var pos span[float32]
				var pp span[float32]
				var pp1 span[float32]
				var i int
				clear(tempbuff2[0:4])
				copy(tempbuff2[4:4+lMem], mem.slice(lMem))
				clear(tempbuff2[lMem+4 : lMem+4+5])
				k = 2*(index-baseSize-(lMem-cbveclen+1)) + cbveclen
				sFilt = lMem - k
				memInd = sFilt + 1 - 4
				pos = span[float32]{data: tmpbuf[:], off: sFilt}
				clear(pos.slice(k))
				for i = 0; i < k; i++ {
					pp = span[float32]{data: tempbuff2[:], off: memInd + i + 4}
					pp1 = span[float32]{data: cbfiltersTbl[:], off: 7}
					for j = 0; j < 8; j++ {
						*pos.at(0) += float32(*pp.at(0) * *pp1.at(0))
						pp = pp.add(1)
						pp1 = pp1.add(-1)
					}
					pos = pos.add(1)
				}
				ihigh = k / 2
				ilow = ihigh - 5
				copy(cbvec.slice(ilow), tmpbuf[lMem+-(k/2):lMem+-(k/2)+ilow])
				alfa1 = float32(0.2)
				alfa = float32(0)
				for j = ilow; j < ihigh; j++ {
					*cbvec.at(j) = float32((float32(1)-alfa)*tmpbuf[lMem-k/2+j]) + float32(alfa*tmpbuf[lMem-k+j])
					alfa += alfa1
				}
				copy(cbvec.add(ihigh).slice(cbveclen-ihigh), tmpbuf[lMem+-k+ihigh:lMem+-k+ihigh+(cbveclen-ihigh)])
			}
		}
	}
}
