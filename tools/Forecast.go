package tools

import (
    "math"
    "gonum.org/v1/gonum/mat"
)

//Returns 2*K features for period P at time t (0-indexed)
func FourierFeatures(t int, P float64, K int) []float64 {
    feats := make([]float64, 0, 2*K)
    x := 2.0 * math.Pi * float64(t) / P
    for k := 1; k <= K; k++ {
        w := float64(k) * x
        feats = append(feats, math.Sin(w), math.Cos(w))
    }
    return feats
}

//Solves (X^T X) beta = X^T y via naive Gaussian elimination.
func SolveNormalEq(X [][]float64, y []float64) ([]float64, error) {
    n, p := len(X), len(X[0])

    // Build matrices
    Xm := mat.NewDense(n, p, nil)
    ym := mat.NewVecDense(n, y)

    for i := 0; i < n; i++ {
        for j := 0; j < p; j++ {
            Xm.Set(i, j, X[i][j])
        }
    }

    // Compute XtX = XᵀX and Xty = Xᵀy
    var Xt mat.Dense
    Xt.CloneFrom(Xm.T())
    var XtX mat.Dense
    XtX.Mul(&Xt, Xm)       // XtX = XᵀX
    var Xty mat.VecDense
    Xty.MulVec(&Xt, ym)    // Xty = Xᵀy

    // Solve (XtX) β = Xty
    var beta mat.VecDense
    err := beta.SolveVec(&XtX, &Xty)
    if err != nil {
        return nil, err
    }
    return beta.RawVector().Data, nil
}
