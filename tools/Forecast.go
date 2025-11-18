package tools

import (
    "math"
)

/**
Auxiliary methods for price forecasting
**/

func rmse(a, b []float64) float64 {
    s := 0.0
    for i := range a {
        d := a[i] - b[i]
        s += d * d
    }
    return math.Sqrt(s / float64(len(a)))
}

// Holt–Winters with damping
func holtDamped(data []float64, alpha, beta, phi float64, steps int) []float64 {
    level := data[0]
    trend := data[1] - data[0]

    for t := 1; t < len(data); t++ {
        prev := level
        level = alpha*data[t] + (1-alpha)*(level+phi*trend)
        trend = beta*(level-prev) + (1-beta)*phi*trend
    }

    out := make([]float64, steps)
    for h := 1; h <= steps; h++ {
        var mult float64
        if math.Abs(1-phi) < 1e-12 {
            mult = float64(h)
        } else {
            mult = phi * (1 - math.Pow(phi, float64(h))) / (1 - phi)
        }
        out[h-1] = level + mult*trend
    }

    return out
}

//Tune Holt parameters alpha, beta, phi
func tuneHoltDamped(trend []float64, holdout int) (bestA, bestB, bestPhi float64) {
    train := trend[:len(trend)-holdout]
    valid := trend[len(trend)-holdout:]

    bestRMSE := math.Inf(1)

    for alpha := 0.2; alpha <= 0.95; alpha += 0.05 {
        for beta := 0.05; beta <= 0.5; beta += 0.05 {
            for phi := 0.80; phi <= 1.00; phi += 0.05 {

                f := holtDamped(train, alpha, beta, phi, holdout)
                r := rmse(f, valid)

                if r < bestRMSE {
                    bestRMSE = r
                    bestA = alpha
                    bestB = beta
                    bestPhi = phi
                }
            }
        }
    }

    if bestA == 0 {
        bestA, bestB, bestPhi = 0.8, 0.2, 0.9
    }

    return
}

func fitAR1(r []float64) float64 {
    var num, den float64
    for i := 1; i < len(r); i++ {
        num += r[i] * r[i-1]
        den += r[i-1] * r[i-1]
    }
    if den == 0 {
        return 0
    }
    return num / den
}

func forecastAR1(phi float64, last float64, steps int) []float64 {
    out := make([]float64, steps)
    val := last
    for i := 0; i < steps; i++ {
        val = phi * val
        out[i] = val
    }
    return out
}
func ForecastBest(priceSeries, trend, seasonal []float64, period int, daysFuture int) float64 {

    n := len(priceSeries)
    holdout := 7
    if n < 20 {
        holdout = 3
    }

    //Tune Holt
    alpha, beta, phi := tuneHoltDamped(trend, holdout)

    //Forecast
    trendF := holtDamped(trend, alpha, beta, phi, daysFuture)

    //Residual
    resid := make([]float64, n)
    for i := 0; i < n; i++ {
        resid[i] = priceSeries[i] - (trend[i] + seasonal[i])
    }

    //AR1
    phiAR := fitAR1(resid)
    residF := forecastAR1(phiAR, resid[n-1], daysFuture)

    //Final
    sum := 0.0
    for h := 1; h <= daysFuture; h++ {
        t := trendF[h-1]
        s := seasonal[(n+h)%period]
        r := residF[h-1]
        sum += t + s + r
    }

    return sum / float64(daysFuture)
}

