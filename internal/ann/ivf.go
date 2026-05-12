package ann

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/coder/hnsw"
)

type candidate struct {
	pos   int
	score float32
}

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
	if idx.nClusters == 0 || k <= 0 {
		return nil, nil
	}

	if len(query) != Dimensions {
		return nil, fmt.Errorf(
			"query must have %d dimensions, got %d",
			Dimensions,
			len(query),
		)
	}

	q := make([]float32, Dimensions)
	copy(q, query)
	normalize(q)

	nearest := idx.centroids.Search(q, 3)

	if len(nearest) == 0 {
		return nil, nil
	}

	best := make([]candidate, 0, k)

	for _, cluster := range nearest {
		cid := cluster.Key

		start := int(idx.bounds[cid])
		end := int(idx.bounds[cid+1])

		for pos := start; pos < end; pos++ {
			score := dotProduct(idx.vectorAt(pos), q)

			if len(best) < k {
				i := len(best)
				best = best[:len(best)+1]

				for i > 0 && score > best[i-1].score {
					best[i] = best[i-1]
					i--
				}

				best[i] = candidate{pos: pos, score: score}

			} else if score > best[k-1].score {
				i := k - 1

				for i > 0 && score > best[i-1].score {
					best[i] = best[i-1]
					i--
				}

				best[i] = candidate{pos: pos, score: score}
			}
		}
	}

	results := make([]SearchResult, len(best))

	for i, c := range best {
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
