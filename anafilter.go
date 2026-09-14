// Port of RFC 3951 Appendix A: anaFilter.c.
// Copyright (C) The Internet Society (2004). See LICENSE.

package ilbc

func anaFilter(in span[float32], a span[float32], len int, out span[float32], mem span[float32]) {
	var i int
	var j int
	var po span[float32]
	var pi span[float32]
	var pm span[float32]
	var pa span[float32]
	po = out
	for i = 0; i < 10; i++ {
		pi = in.add(i)
		pm = mem.add(9)
		pa = a
		*po.at(0) = float32(0)
		for j = 0; j <= i; j++ {
			*po.at(0) += float32(*pa.at(0) * *pi.at(0))
			pa = pa.add(1)
			pi = pi.add(-1)
		}
		for j = i + 1; j < 11; j++ {
			*po.at(0) += float32(*pa.at(0) * *pm.at(0))
			pa = pa.add(1)
			pm = pm.add(-1)
		}
		po = po.add(1)
	}
	for i = 10; i < len; i++ {
		pi = in.add(i)
		pa = a
		*po.at(0) = float32(0)
		for j = 0; j < 11; j++ {
			*po.at(0) += float32(*pa.at(0) * *pi.at(0))
			pa = pa.add(1)
			pi = pi.add(-1)
		}
		po = po.add(1)
	}
	copy(mem.slice(10), in.add(len-10).slice(10))
}
