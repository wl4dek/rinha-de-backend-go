package ann

import (
	"fmt"
	"math"
	"math/rand/v2"
	"testing"

	"github.com/coder/hnsw"
)

//
// ==========================
// HELPERS
// ==========================
//

func testIndex() *IVFIndex {
	nClusters := 2
	nPerCluster := 5
	nVectors := nClusters * nPerCluster

	bounds := []uint32{0, 5, 10}
	vectors := make([]float32, nVectors*Dimensions)
	labels := make([]byte, nVectors)

	for v := 0; v < nPerCluster; v++ {
		base := v * Dimensions
		vectors[base] = 1
	}

	for v := 0; v < nPerCluster; v++ {
		base := (nPerCluster + v) * Dimensions
		vectors[base] = -1
	}

	labels[0] = 1

	centroids := [][]float32{
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{-1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	}

	g := hnsw.NewGraph[int]()
	g.M = 4
	g.EfSearch = 8
	g.Add(
		hnsw.MakeNode(0, centroids[0]),
		hnsw.MakeNode(1, centroids[1]),
	)

	return &IVFIndex{
		centroids: g,
		labels:    labels,
		bounds:    bounds,
		vectors:   vectors,
		nVectors:  nVectors,
		nClusters: nClusters,
	}
}

func benchIndex() *IVFIndex {
	nClusters := 3
	nPerCluster := 500
	nVectors := nClusters * nPerCluster

	rng := rand.New(rand.NewPCG(42, 99))

	bounds := make([]uint32, nClusters+1)

	for i := range nClusters {
		bounds[i] = uint32(i * nPerCluster)
	}

	bounds[nClusters] = uint32(nVectors)

	vectors := make([]float32, nVectors*Dimensions)
	labels := make([]byte, nVectors)

	centroids := make([][]float32, nClusters)

	for c := range nClusters {
		cent := make([]float32, Dimensions)

		for j := range cent {
			cent[j] = float32(rng.Float64()*2 - 1)
		}

		normalize(cent)
		centroids[c] = cent

		for v := range nPerCluster {
			base := (c*nPerCluster + v) * Dimensions
			vec := vectors[base : base+Dimensions]
			copy(vec, cent)

			for j := range vec {
				vec[j] += float32(rng.Float64()-0.5) * 0.2
			}

			normalize(vec)

			if v%3 == 0 {
				labels[c*nPerCluster+v] = 1
			}
		}
	}

	g := hnsw.NewGraph[int]()
	g.M = 8
	g.EfSearch = 16

	nodes := make([]hnsw.Node[int], nClusters)

	for i, c := range centroids {
		nodes[i] = hnsw.MakeNode(i, c)
	}

	g.Add(nodes...)

	return &IVFIndex{
		centroids: g,
		labels:    labels,
		bounds:    bounds,
		vectors:   vectors,
		nVectors:  nVectors,
		nClusters: nClusters,
	}
}

var benchResult []SearchResult
var benchSink SearchResult

//
// ==========================
// TESTES
// ==========================
//

func TestDotProduct(t *testing.T) {
	tests := []struct {
		name string
		a    []float32
		b    []float32
		want float32
	}{
		{
			name: "unit vectors same direction",
			a:    []float32{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			b:    []float32{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			want: 1,
		},
		{
			name: "unit vectors opposite",
			a:    []float32{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			b:    []float32{-1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			want: -1,
		},
		{
			name: "orthogonal",
			a:    []float32{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			b:    []float32{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			want: 0,
		},
		{
			name: "all dimensions",
			a:    []float32{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
			b:    []float32{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
			want: 14,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dotProduct(tt.a, tt.b)

			if got != tt.want {
				t.Errorf("dotProduct = %f, want %f", got, tt.want)
			}
		})
	}
}

func TestNormalize(t *testing.T) {
	t.Run("non-zero vector", func(t *testing.T) {
		v := []float32{3, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

		normalize(v)

		var norm float64

		for _, x := range v {
			norm += float64(x * x)
		}

		if math.Abs(norm-1) > 1e-6 {
			t.Errorf("norm = %f, want 1", norm)
		}
	})

	t.Run("zero vector stays zero", func(t *testing.T) {
		v := []float32{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

		normalize(v)

		for _, x := range v {
			if x != 0 {
				t.Errorf("expected 0, got %f", x)
			}
		}
	})

	t.Run("all ones", func(t *testing.T) {
		v := []float32{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}

		normalize(v)

		var dot float32

		for i := 0; i < Dimensions; i++ {
			dot += v[i] * v[i]
		}

		if math.Abs(float64(dot-1)) > 1e-6 {
			t.Errorf("norm^2 = %f, want 1", dot)
		}
	})
}

func TestNearestCentroid(t *testing.T) {
	centroids := [][]float32{
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	}

	query := []float32{1, 0.1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	normalize(query)

	got := nearestCentroid(centroids, query)

	if got != 0 {
		t.Errorf("nearestCentroid = %d, want 0", got)
	}
}

func TestNearestCentroidDist(t *testing.T) {
	centroids := [][]float32{
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{-1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	}

	query := []float32{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	normalize(query)

	got, dist := nearestCentroidDist(centroids, query)

	if got != 0 {
		t.Errorf("nearestCentroidDist idx = %d, want 0", got)
	}

	if math.Abs(float64(dist)) > 1e-6 {
		t.Errorf("nearestCentroidDist dist = %f, want 0", dist)
	}
}

func TestSearch(t *testing.T) {
	idx := testIndex()

	query := []float32{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

	results, err := idx.Search(query, 10)
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 10 {
		t.Fatalf("len(results) = %d, want 10", len(results))
	}

	for i, r := range results[:5] {
		if r.Dist > 0.01 {
			t.Errorf("results[%d].Dist = %f, want ~0 (cluster 0)", i, r.Dist)
		}
	}

	for i, r := range results[5:] {
		if r.Dist < 1.99 {
			t.Errorf("results[%d].Dist = %f, want ~2 (cluster 1)", i+5, r.Dist)
		}
	}

	if results[0].Label != "fraud" {
		t.Errorf("results[0].Label = %s, want fraud", results[0].Label)
	}
}

func TestSearchEmpty(t *testing.T) {
	idx := &IVFIndex{nClusters: 0}

	results, err := idx.Search([]float32{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, 5)
	if err != nil {
		t.Fatal(err)
	}

	if results != nil {
		t.Errorf("expected nil, got %v", results)
	}
}

func TestSearchKZero(t *testing.T) {
	idx := testIndex()

	results, err := idx.Search([]float32{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, 0)
	if err != nil {
		t.Fatal(err)
	}

	if results != nil {
		t.Errorf("expected nil, got %v", results)
	}
}

func TestSearchPreservesQuery(t *testing.T) {
	idx := testIndex()

	original := []float32{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	query := make([]float32, Dimensions)
	copy(query, original)

	_, _ = idx.Search(query, 5)

	for i := range query {
		if query[i] != original[i] {
			t.Errorf("query[%d] = %f, want %f (was modified)", i, query[i], original[i])
		}
	}
}

func TestSearchWrongDimensions(t *testing.T) {
	idx := testIndex()

	_, err := idx.Search([]float32{1, 2, 3}, 5)
	if err == nil {
		t.Fatal("expected error for wrong dimensions, got nil")
	}
}

//
// ==========================
// BENCHMARKS
// ==========================
//

func BenchmarkDotProduct(b *testing.B) {
	rng := rand.New(rand.NewPCG(1, 2))

	a := make([]float32, Dimensions)
	bv := make([]float32, Dimensions)

	for i := range a {
		a[i] = float32(rng.Float64())
		bv[i] = float32(rng.Float64())
	}

	b.ResetTimer()

	var sum float32

	for i := 0; i < b.N; i++ {
		sum += dotProduct(a, bv)
	}

	benchSink.Dist = sum
}

func BenchmarkNormalize(b *testing.B) {
	rng := rand.New(rand.NewPCG(1, 2))

	v := make([]float32, Dimensions)

	for i := range v {
		v[i] = float32(rng.Float64()*2 - 1)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		normalize(v)
	}
}

func BenchmarkSearch(b *testing.B) {
	idx := benchIndex()

	rng := rand.New(rand.NewPCG(1, 2))

	queries := make([][]float32, 100)

	for i := range queries {
		q := make([]float32, Dimensions)

		for j := range q {
			q[j] = float32(rng.Float64()*2 - 1)
		}

		queries[i] = q
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var err error

		benchResult, err = idx.Search(queries[i%len(queries)], 5)

		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSearchK(b *testing.B) {
	idx := benchIndex()

	rng := rand.New(rand.NewPCG(1, 2))

	queries := make([][]float32, 100)

	for i := range queries {
		q := make([]float32, Dimensions)

		for j := range q {
			q[j] = float32(rng.Float64()*2 - 1)
		}

		queries[i] = q
	}

	for _, k := range []int{1, 5, 10, 20, 50, 100} {
		b.Run(fmt.Sprintf("k=%d", k), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				var err error

				benchResult, err = idx.Search(queries[i%len(queries)], k)

				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkSearchClusterSize(b *testing.B) {
	for _, nPerCluster := range []int{100, 500, 1000} {
		b.Run(fmt.Sprintf("cluster=%d", nPerCluster), func(b *testing.B) {
			nClusters := 3
			nVectors := nClusters * nPerCluster

			rng := rand.New(rand.NewPCG(42, uint64(nPerCluster)))

			bounds := make([]uint32, nClusters+1)

			for i := range nClusters {
				bounds[i] = uint32(i * nPerCluster)
			}

			bounds[nClusters] = uint32(nVectors)

			vectors := make([]float32, nVectors*Dimensions)
			labels := make([]byte, nVectors)

			centroids := make([][]float32, nClusters)

			for c := range nClusters {
				cent := make([]float32, Dimensions)

				for j := range cent {
					cent[j] = float32(rng.Float64()*2 - 1)
				}

				normalize(cent)
				centroids[c] = cent

				for v := range nPerCluster {
					base := (c*nPerCluster + v) * Dimensions
					vec := vectors[base : base+Dimensions]
					copy(vec, cent)

					for j := range vec {
						vec[j] += float32(rng.Float64()-0.5) * 0.2
					}

					normalize(vec)

					if v%3 == 0 {
						labels[c*nPerCluster+v] = 1
					}
				}
			}

			g := hnsw.NewGraph[int]()
			g.M = 8
			g.EfSearch = 16

			nodes := make([]hnsw.Node[int], nClusters)

			for i, c := range centroids {
				nodes[i] = hnsw.MakeNode(i, c)
			}

			g.Add(nodes...)

			idx := &IVFIndex{
				centroids: g,
				labels:    labels,
				bounds:    bounds,
				vectors:   vectors,
				nVectors:  nVectors,
				nClusters: nClusters,
			}

			qbatch := make([][]float32, 50)

			for i := range qbatch {
				q := make([]float32, Dimensions)

				for j := range q {
					q[j] = float32(rng.Float64()*2 - 1)
				}

				qbatch[i] = q
			}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				var err error

				benchResult, err = idx.Search(qbatch[i%len(qbatch)], 5)

				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
