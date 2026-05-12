package ann

import "math"

func normalize(v []float32) {
	var norm float32

	for i := 0; i < Dimensions; i++ {
		norm += v[i] * v[i]
	}

	if norm == 0 {
		return
	}

	inv := 1 / float32(math.Sqrt(float64(norm)))

	for i := 0; i < Dimensions; i++ {
		v[i] *= inv
	}
}

func dotProduct(a, b []float32) float32 {
	return a[0]*b[0] +
		a[1]*b[1] +
		a[2]*b[2] +
		a[3]*b[3] +
		a[4]*b[4] +
		a[5]*b[5] +
		a[6]*b[6] +
		a[7]*b[7] +
		a[8]*b[8] +
		a[9]*b[9] +
		a[10]*b[10] +
		a[11]*b[11] +
		a[12]*b[12] +
		a[13]*b[13]
}

func nearestCentroid(centroids [][]float32, vec []float32) int {
	best := 0
	bestScore := float32(-2)

	for i, c := range centroids {
		score := dotProduct(c, vec)

		if score > bestScore {
			bestScore = score
			best = i
		}
	}

	return best
}

func nearestCentroidDist(centroids [][]float32, vec []float32) (int, float32) {
	best := 0
	bestDist := float32(2)

	for i, c := range centroids {
		dist := 1 - dotProduct(c, vec)

		if dist < bestDist {
			bestDist = dist
			best = i
		}
	}

	return best, bestDist
}

func addVec(to, from []float32) {
	for i := 0; i < Dimensions; i++ {
		to[i] += from[i]
	}
}
