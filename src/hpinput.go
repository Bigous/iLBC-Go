// Port of RFC 3951 Appendix A: hpInput.c.
// Copyright (C) The Internet Society (2004). See LICENSE.

package ilbc

func hpInput(in span[float32], len int, out span[float32], mem span[float32]) {
	var i int
	var pi span[float32]
	var po span[float32]
	pi = in.add(0)
	po = out.add(0)
	for i = 0; i < len; i++ {
		*po.at(0) = float32(hpiZeroCoefsTbl[0] * *pi.at(0))
		*po.at(0) += float32(hpiZeroCoefsTbl[1] * *mem.at(0))
		*po.at(0) += float32(hpiZeroCoefsTbl[2] * *mem.at(1))
		*mem.at(1) = *mem.at(0)
		*mem.at(0) = *pi.at(0)
		po = po.add(1)
		pi = pi.add(1)
	}
	po = out.add(0)
	for i = 0; i < len; i++ {
		*po.at(0) -= float32(hpiPoleCoefsTbl[1] * *mem.at(2))
		*po.at(0) -= float32(hpiPoleCoefsTbl[2] * *mem.at(3))
		*mem.at(3) = *mem.at(2)
		*mem.at(2) = *po.at(0)
		po = po.add(1)
	}
}
