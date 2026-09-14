// Port of RFC 3951 Appendix A: lsf.c.
// Copyright (C) The Internet Society (2004). See LICENSE.

package ilbc

import "math"

func a2lsf(freq span[float32], a span[float32]) {
	var steps [4]float32
	steps = [4]float32{float32(0.00635), float32(0.003175), float32(0.0015875), float32(0.00079375)}
	var step float32
	var stepIdx int
	var lspIndex int
	var p [5]float32
	var q [5]float32
	var pPre [5]float32
	var qPre [5]float32
	var oldP [1]float32
	var oldQ [1]float32
	var old span[float32]
	var pqCoef span[float32]
	var omega float32
	var oldOmega float32
	var i int
	var hlp float32
	var hlp1 float32
	var hlp2 float32
	var hlp3 float32
	var hlp4 float32
	var hlp5 float32
	for i = 0; i < 5; i++ {
		p[i] = float32(float32(-1) * (*a.at(i + 1) + *a.at(10 - i)))
		q[i] = *a.at(10 - i) - *a.at(i + 1)
	}
	pPre[0] = float32(-1) - p[0]
	pPre[1] = -pPre[0] - p[1]
	pPre[2] = -pPre[1] - p[2]
	pPre[3] = -pPre[2] - p[3]
	pPre[4] = -pPre[3] - p[4]
	pPre[4] = pPre[4] / float32(2)
	qPre[0] = float32(1) - q[0]
	qPre[1] = qPre[0] - q[1]
	qPre[2] = qPre[1] - q[2]
	qPre[3] = qPre[2] - q[3]
	qPre[4] = qPre[3] - q[4]
	qPre[4] = qPre[4] / float32(2)
	omega = float32(0)
	oldOmega = float32(0)
	oldP[0] = float32(1e+37)
	oldQ[0] = float32(1e+37)
	for lspIndex = 0; lspIndex < 10; lspIndex++ {
		if lspIndex&1 == 0 {
			pqCoef = span[float32]{data: pPre[:]}
			old = span[float32]{data: oldP[:]}
		} else {
			pqCoef = span[float32]{data: qPre[:]}
			old = span[float32]{data: oldQ[:]}
		}
		{
			stepIdx = 0
			step = steps[stepIdx]
			for stepIdx < 4 {
				hlp = float32(math.Cos(float64(float32(omega * float32(6.2831855)))))
				hlp1 = float32(float32(2)*hlp) + *pqCoef.at(0)
				hlp2 = float32(float32(float32(2)*hlp)*hlp1) - float32(1) + *pqCoef.at(1)
				hlp3 = float32(float32(float32(2)*hlp)*hlp2) - hlp1 + *pqCoef.at(2)
				hlp4 = float32(float32(float32(2)*hlp)*hlp3) - hlp2 + *pqCoef.at(3)
				hlp5 = float32(hlp*hlp4) - hlp3 + *pqCoef.at(4)
				if float64(float32(hlp5**old.at(0))) <= float64(0) || float64(omega) >= float64(0.5) {
					if stepIdx == 3 {
						if math.Abs(float64(hlp5)) >= math.Abs(float64(*old.at(0))) {
							*freq.at(lspIndex) = omega - step
						} else {
							*freq.at(lspIndex) = omega
						}
						if float64(*old.at(0)) >= float64(0) {
							*old.at(0) = float32(-1e+37)
						} else {
							*old.at(0) = float32(1e+37)
						}
						omega = oldOmega
						stepIdx = 0
						stepIdx = 4
					} else {
						if stepIdx == 0 {
							oldOmega = omega
						}
						stepIdx++
						omega -= steps[stepIdx]
						step = steps[stepIdx]
					}
				} else {
					*old.at(0) = hlp5
					omega += step
				}
			}
		}
	}
	for i = 0; i < 10; i++ {
		*freq.at(i) = float32(*freq.at(i) * float32(6.2831855))
	}
}

func lsf2a(aCoef span[float32], freq span[float32]) {
	var i int
	var j int
	var hlp float32
	var p [5]float32
	var q [5]float32
	var a [6]float32
	var a1 [5]float32
	var a2 [5]float32
	var b [6]float32
	var b1 [5]float32
	var b2 [5]float32
	for i = 0; i < 10; i++ {
		*freq.at(i) = float32(*freq.at(i) * float32(0.15915494))
	}
	if float64(*freq.at(0)) <= float64(0) || float64(*freq.at(9)) >= float64(0.5) {
		if float64(*freq.at(0)) <= float64(0) {
			*freq.at(0) = float32(0.022)
		}
		if float64(*freq.at(9)) >= float64(0.5) {
			*freq.at(9) = float32(0.499)
		}
		hlp = (*freq.at(9) - *freq.at(0)) / float32(9)
		for i = 1; i < 10; i++ {
			*freq.at(i) = *freq.at(i - 1) + hlp
		}
	}
	clear(a1[0:5])
	clear(a2[0:5])
	clear(b1[0:5])
	clear(b2[0:5])
	clear(a[0:6])
	clear(b[0:6])
	for i = 0; i < 5; i++ {
		p[i] = float32(math.Cos(float64(float32(float32(6.2831855) * *freq.at(2 * i)))))
		q[i] = float32(math.Cos(float64(float32(float32(6.2831855) * *freq.at(2*i + 1)))))
	}
	a[0] = float32(0.25)
	b[0] = float32(0.25)
	for i = 0; i < 5; i++ {
		a[i+1] = a[i] - float32(float32(float32(2)*p[i])*a1[i]) + a2[i]
		b[i+1] = b[i] - float32(float32(float32(2)*q[i])*b1[i]) + b2[i]
		a2[i] = a1[i]
		a1[i] = a[i]
		b2[i] = b1[i]
		b1[i] = b[i]
	}
	for j = 0; j < 10; j++ {
		if j == 0 {
			a[0] = float32(0.25)
			b[0] = float32(-0.25)
		} else {
			a[0] = float32(0)
			b[0] = float32(0)
		}
		for i = 0; i < 5; i++ {
			a[i+1] = a[i] - float32(float32(float32(2)*p[i])*a1[i]) + a2[i]
			b[i+1] = b[i] - float32(float32(float32(2)*q[i])*b1[i]) + b2[i]
			a2[i] = a1[i]
			a1[i] = a[i]
			b2[i] = b1[i]
			b1[i] = b[i]
		}
		*aCoef.at(j + 1) = float32(float32(2) * (a[5] + b[5]))
	}
	*aCoef.at(0) = float32(1)
}
