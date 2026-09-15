// Port of RFC 3951 Appendix A: helpfun.c.
// Copyright (C) The Internet Society (2004). See LICENSE.

package ilbc

func autocorr(r span[float32], x span[float32], length int, order int) {
	var lag int
	var n int
	var sum float32
	for lag = 0; lag <= order; lag++ {
		sum = float32(0)
		for n = 0; n < length-lag; n++ {
			sum += float32(*x.at(n) * *x.at(n + lag))
		}
		*r.at(lag) = sum
	}
}

func window(z span[float32], x span[float32], y span[float32], length int) {
	var i int
	for i = 0; i < length; i++ {
		*z.at(i) = float32(*x.at(i) * *y.at(i))
	}
}

func levdurb(a span[float32], k span[float32], r span[float32], order int) {
	var sum float32
	var alpha float32
	var m int
	var mH int
	var i int
	*a.at(0) = float32(1)
	if *r.at(0) < float32(2.220446e-16) {
		for i = 0; i < order; i++ {
			*k.at(i) = float32(0)
			*a.at(i + 1) = float32(0)
		}
	} else {
		*a.at(1) = -*r.at(1) / *r.at(0)
		*k.at(0) = -*r.at(1) / *r.at(0)
		alpha = *r.at(0) + float32(*r.at(1)**k.at(0))
		for m = 1; m < order; m++ {
			sum = *r.at(m + 1)
			for i = 0; i < m; i++ {
				sum += float32(*a.at(i + 1) * *r.at(m - i))
			}
			*k.at(m) = -sum / alpha
			alpha += float32(*k.at(m) * sum)
			mH = (m + 1) >> 1
			for i = 0; i < mH; i++ {
				sum = *a.at(i + 1) + float32(*k.at(m)**a.at(m - i))
				*a.at(m - i) += float32(*k.at(m) * *a.at(i + 1))
				*a.at(i + 1) = sum
			}
			*a.at(m + 1) = *k.at(m)
		}
	}
}

func interpolate(out span[float32], in1 span[float32], in2 span[float32], coef float32, length int) {
	var i int
	var invcoef float32
	invcoef = float32(1) - coef
	for i = 0; i < length; i++ {
		*out.at(i) = float32(coef**in1.at(i)) + float32(invcoef**in2.at(i))
	}
}

func bwexpand(out span[float32], input span[float32], coef float32, length int) {
	var i int
	var chirp float32
	chirp = coef
	*out.at(0) = *input.at(0)
	for i = 1; i < length; i++ {
		*out.at(i) = float32(chirp * *input.at(i))
		chirp *= coef
	}
}

func vq(xq span[float32], index span[int], cB span[float32], x span[float32], nCb int, dim int) {
	var i int
	var j int
	var pos int
	var minindex int
	var dist float32
	var tmp float32
	var mindist float32
	pos = 0
	mindist = float32(1e+37)
	minindex = 0
	for j = 0; j < nCb; j++ {
		dist = *x.at(0) - *cB.at(pos)
		dist *= dist
		for i = 1; i < dim; i++ {
			tmp = *x.at(i) - *cB.at(pos + i)
			dist += float32(tmp * tmp)
		}
		if dist < mindist {
			mindist = dist
			minindex = j
		}
		pos += dim
	}
	for i = 0; i < dim; i++ {
		*xq.at(i) = *cB.at(minindex*dim + i)
	}
	*index.at(0) = minindex
}

func splitVQ(qX span[float32], index span[int], x span[float32], cB span[float32], nsplit int, dim span[int], cbsize span[int]) {
	var cbPos int
	var xPos int
	var i int
	cbPos = 0
	xPos = 0
	for i = 0; i < nsplit; i++ {
		vq(qX.add(xPos), index.add(i), cB.add(cbPos), x.add(xPos), *cbsize.at(i), *dim.at(i))
		xPos += *dim.at(i)
		cbPos += *dim.at(i) * *cbsize.at(i)
	}
}

func sortSq(xq span[float32], index span[int], x float32, cb span[float32], cbSize int) {
	var i int
	if x <= *cb.at(0) {
		*index.at(0) = 0
		*xq.at(0) = *cb.at(0)
	} else {
		i = 0
		for x > *cb.at(i) && i < cbSize-1 {
			i++
		}
		if x > (*cb.at(i)+*cb.at(i - 1))/float32(2) {
			*index.at(0) = i
			*xq.at(0) = *cb.at(i)
		} else {
			*index.at(0) = i - 1
			*xq.at(0) = *cb.at(i - 1)
		}
	}
}

func checkLSF(lsf span[float32], dim int, noAn int) int {
	var k int
	var n int
	var m int
	var nit int
	nit = 2
	var change int
	change = 0
	var pos int
	var eps float32
	eps = float32(0.039)
	var eps2 float32
	eps2 = float32(0.0195)
	var maxlsf float32
	maxlsf = float32(3.14)
	var minlsf float32
	minlsf = float32(0.01)
	for n = 0; n < nit; n++ {
		for m = 0; m < noAn; m++ {
			for k = 0; k < dim-1; k++ {
				pos = m*dim + k
				if *lsf.at(pos + 1)-*lsf.at(pos) < eps {
					if *lsf.at(pos + 1) < *lsf.at(pos) {

						*lsf.at(pos + 1) = *lsf.at(pos) + eps2
						*lsf.at(pos) = *lsf.at(pos + 1) - eps2
					} else {
						*lsf.at(pos) -= eps2
						*lsf.at(pos + 1) += eps2
					}
					change = 1
				}
				if *lsf.at(pos) < minlsf {
					*lsf.at(pos) = minlsf
					change = 1
				}
				if *lsf.at(pos) > maxlsf {
					*lsf.at(pos) = maxlsf
					change = 1
				}
			}
		}
	}
	return change
}
