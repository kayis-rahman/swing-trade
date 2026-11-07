package strategy

import (
	"fmt"
	"math"
)

const eulerMascheroni = 0.5772156649015329

// ComputeDeflatedSharpeRatio mirrors Java's DeflatedSharpeRatio (plan §6.4).
//
// It returns the probability that the true Sharpe ratio exceeds zero after
// correcting for selection bias across numTrials independent trials.
func ComputeDeflatedSharpeRatio(observedSharpe float64, numTrials, numObservations int, sharpeStdError float64) (float64, error) {
	if numTrials < 1 {
		return 0, fmt.Errorf("numTrials must be >= 1")
	}
	if numObservations <= 1 {
		return 0, fmt.Errorf("numObservations must be > 1")
	}
	stdError := sharpeStdError
	if !(stdError > 0) {
		stdError = math.Max(0.1, math.Abs(observedSharpe)*0.5+0.1)
	}

	var expectedMaxSharpe float64
	if numTrials > 1 {
		expectedMaxSharpe = stdError * ((1-eulerMascheroni)*inverseNormalCdf(1.0-1.0/float64(numTrials)) +
			eulerMascheroni*inverseNormalCdf(1.0-1.0/(float64(numTrials)*math.E)))
	}

	z := (observedSharpe - expectedMaxSharpe) * math.Sqrt(float64(numObservations-1))
	return normalCdf(z), nil
}

// normalCdf is the standard normal CDF via the Abramowitz-Stegun erf approximation.
func normalCdf(x float64) float64 {
	return 0.5 * (1.0 + erf(x/math.Sqrt(2.0)))
}

// inverseNormalCdf is Acklam's-style approximation via bisection, matching Java.
func inverseNormalCdf(p float64) float64 {
	if p <= 0.0 {
		return math.Inf(-1)
	}
	if p >= 1.0 {
		return math.Inf(1)
	}
	lo := -10.0
	hi := 10.0
	for i := 0; i < 100; i++ {
		mid := (lo + hi) / 2.0
		if normalCdf(mid) < p {
			lo = mid
		} else {
			hi = mid
		}
	}
	return (lo + hi) / 2.0
}

func erf(x float64) float64 {
	// Abramowitz & Stegun 7.1.26, max error ~1.5e-7.
	t := 1.0 / (1.0 + 0.3275911*math.Abs(x))
	a := 1.061405429*t - 1.453152027
	b := a*t + 1.421413741
	c := b*t - 0.284496736
	d := c*t + 0.254829592
	y := 1.0 - d*t*math.Exp(-x*x)
	if x >= 0 {
		return y
	}
	return -y
}
