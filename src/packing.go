// Port of RFC 3951 Appendix A: packing.c.
// Copyright (C) The Internet Society (2004). See LICENSE.

package ilbc

func packsplit(index span[int], firstpart span[int], rest span[int], bitnoFirstpart int, bitnoTotal int) {
	var bitnoRest int
	bitnoRest = bitnoTotal - bitnoFirstpart
	*firstpart.at(0) = *index.at(0) >> bitnoRest
	*rest.at(0) = *index.at(0) - *firstpart.at(0)<<bitnoRest
}

func packcombine(index span[int], rest int, bitnoRest int) {
	*index.at(0) = *index.at(0) << bitnoRest
	*index.at(0) += rest
}

func dopack(bitstream span[span[byte]], index int, bitno int, pos span[int]) {
	var posLeft int
	if *pos.at(0) == 0 {
		*bitstream.at(0).at(0) = 0
	}
	for bitno > 0 {
		if *pos.at(0) == 8 {
			*pos.at(0) = 0
			*bitstream.at(0) = bitstream.at(0).add(1)
			*bitstream.at(0).at(0) = 0
		}
		posLeft = 8 - *pos.at(0)
		if bitno <= posLeft {
			*bitstream.at(0).at(0) |= byte(index << (posLeft - bitno))
			*pos.at(0) += bitno
			bitno = 0
		} else {
			*bitstream.at(0).at(0) |= byte(index >> (bitno - posLeft))
			*pos.at(0) = 8
			index -= index >> (bitno - posLeft) << (bitno - posLeft)
			bitno -= posLeft
		}
	}
}

func unpack(bitstream span[span[byte]], index span[int], bitno int, pos span[int]) {
	var bitsLeft int
	*index.at(0) = 0
	for bitno > 0 {
		if *pos.at(0) == 8 {
			*pos.at(0) = 0
			*bitstream.at(0) = bitstream.at(0).add(1)
		}
		bitsLeft = 8 - *pos.at(0)
		if bitsLeft >= bitno {
			*index.at(0) += int(*bitstream.at(0).at(0)) << *pos.at(0) & 255 >> (8 - bitno)
			*pos.at(0) += bitno
			bitno = 0
		} else {
			if 8-bitno > 0 {
				*index.at(0) += int(*bitstream.at(0).at(0)) << *pos.at(0) & 255 >> (8 - bitno)
				*pos.at(0) = 8
			} else {
				*index.at(0) += int(*bitstream.at(0).at(0)) << *pos.at(0) & 255 << (bitno - 8)
				*pos.at(0) = 8
			}
			bitno -= bitsLeft
		}
	}
}
