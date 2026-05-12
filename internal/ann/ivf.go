package ann

import (
	"container/heap"
	"encoding/binary"
	"log"
	"os"
	"path/filepath"

	"github.com/coder/hnsw"
)

type IVFIndex struct {
	centroids *hnsw.Graph[int]
	labels    []byte
	bounds    []uint32
	vectors   []float32
	nVectors  int
	nClusters int
}

func (idx *IVFIndex) vectorAt(pos int) []float32 {
	start := pos * Dimensions
	return idx.vectors[start : start+Dimensions]
}

func (idx *IVFIndex) labelFor(pos int) string {
	if idx.labels[pos] == 1 {
		return "fraud"
	}
	return "legit"
}

func (idx *IVFIndex) Search(query []float32, k int) ([]SearchResult, error) {
	if idx.nClusters == 0 {
		return nil, nil
	}

	normalize(query)

	nearest := idx.centroids.Search(query, 3)

	if len(nearest) == 0 {
		return nil, nil
	}

	h := make(maxHeap, 0, k)

	for _, cluster := range nearest {
		cid := cluster.Key

		start := int(idx.bounds[cid])
		end := int(idx.bounds[cid+1])

		for pos := start; pos < end; pos++ {
			score := dotProduct(idx.vectorAt(pos), query)

			if len(h) < k {
				heap.Push(&h, candidate{pos: pos, score: score})
				continue
			}

			if score > h[0].score {
				h[0] = candidate{pos: pos, score: score}
				heap.Fix(&h, 0)
			}
		}
	}

	results := make([]SearchResult, len(h))

	for i := len(h) - 1; i >= 0; i-- {
		c := heap.Pop(&h).(candidate)

		results[i] = SearchResult{
			Label: idx.labelFor(c.pos),
			Dist:  1 - c.score,
		}
	}

	return results, nil
}

func LoadIVF(path string) (*IVFIndex, error) {
	hdr, err := os.ReadFile(filepath.Join(path, "ivf.bin"))
	if err != nil {
		return nil, err
	}

	nVectors := int(binary.LittleEndian.Uint32(hdr[0:4]))
	nClusters := int(binary.LittleEndian.Uint32(hdr[4:8]))

	log.Printf("Loading IVF index: vectors=%d clusters=%d", nVectors, nClusters)

	labels, err := os.ReadFile(filepath.Join(path, "labels.bin"))
	if err != nil {
		return nil, err
	}

	bounds, err := readUint32File(filepath.Join(path, "cluster_bounds.bin"))
	if err != nil {
		return nil, err
	}

	vectors, err := mmapFloat32(filepath.Join(path, "vectors.bin"))
	if err != nil {
		return nil, err
	}

	centroidsRaw, err := readFloat32Vectors(filepath.Join(path, "centroids.bin"), nClusters)
	if err != nil {
		return nil, err
	}

	g := hnsw.NewGraph[int]()
	g.M = 16
	g.EfSearch = 32

	nodes := make([]hnsw.Node[int], nClusters)

	for i, vec := range centroidsRaw {
		nodes[i] = hnsw.MakeNode(i, vec)
	}

	g.Add(nodes...)

	maxCluster := 0

	for i := 0; i < nClusters; i++ {
		size := int(bounds[i+1] - bounds[i])

		if size > maxCluster {
			maxCluster = size
		}
	}

	log.Printf("max cluster size=%d", maxCluster)

	return &IVFIndex{
		centroids: g,
		labels:    labels,
		bounds:    bounds,
		vectors:   vectors,
		nVectors:  nVectors,
		nClusters: nClusters,
	}, nil
}
