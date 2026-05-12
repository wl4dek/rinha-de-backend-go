package ann

import (
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
)

func BuildIVF(refs []Reference, nClusters int, outputPath string) error {
	nVectors := len(refs)

	if nVectors == 0 {
		return fmt.Errorf("no references")
	}

	log.Printf("Building IVF vectors=%d clusters=%d", nVectors, nClusters)

	for i := range refs {
		normalize(refs[i].Vector)
	}

	sampleSize := min(50000, nVectors)

	perm := rand.Perm(nVectors)

	sample := make([][]float32, sampleSize)

	for i, idx := range perm[:sampleSize] {
		sample[i] = refs[idx].Vector
	}

	centroids := kmeansPP(sample, nClusters)

	for iter := 0; iter < 20; iter++ {
		assignments := make([]int, sampleSize)

		for i, vec := range sample {
			assignments[i] = nearestCentroid(centroids, vec)
		}

		newCentroids := make([][]float32, nClusters)
		counts := make([]int, nClusters)

		for i := range newCentroids {
			newCentroids[i] = make([]float32, Dimensions)
		}

		for i, cid := range assignments {
			counts[cid]++
			addVec(newCentroids[cid], sample[i])
		}

		for i := range newCentroids {
			if counts[i] == 0 {
				copy(newCentroids[i], centroids[i])
				continue
			}

			inv := 1 / float32(counts[i])

			for j := 0; j < Dimensions; j++ {
				newCentroids[i][j] *= inv
			}

			normalize(newCentroids[i])
		}

		var shift float32

		for i := range newCentroids {
			for j := 0; j < Dimensions; j++ {
				d := newCentroids[i][j] - centroids[i][j]
				shift += d * d
			}
		}

		copy(centroids, newCentroids)

		if shift < 1e-6 {
			log.Printf("kmeans converged iter=%d", iter)
			break
		}
	}

	clusterOf := make([]uint16, nVectors)

	for i, ref := range refs {
		clusterOf[i] = uint16(nearestCentroid(centroids, ref.Vector))
	}

	clusterCounts := make([]uint32, nClusters)

	for _, cid := range clusterOf {
		clusterCounts[cid]++
	}

	bounds := make([]uint32, nClusters+1)

	var off uint32

	for i := range clusterCounts {
		bounds[i] = off
		off += clusterCounts[i]
	}

	bounds[nClusters] = off

	order := make([]int, nVectors)

	cursor := make([]uint32, nClusters)

	copy(cursor, bounds[:nClusters])

	for i := range refs {
		cid := clusterOf[i]
		order[cursor[cid]] = i
		cursor[cid]++
	}

	write := func(name string, data []byte) error {
		return os.WriteFile(filepath.Join(outputPath, name), data, 0644)
	}

	vecData := make([]byte, nVectors*Dimensions*4)

	for dstIdx, srcIdx := range order {
		base := dstIdx * Dimensions * 4

		for j, v := range refs[srcIdx].Vector {
			binary.LittleEndian.PutUint32(vecData[base+j*4:], math.Float32bits(v))
		}
	}

	if err := write("vectors.bin", vecData); err != nil {
		return err
	}

	labData := make([]byte, nVectors)

	for dstIdx, srcIdx := range order {
		if refs[srcIdx].Label == "fraud" {
			labData[dstIdx] = 1
		}
	}

	if err := write("labels.bin", labData); err != nil {
		return err
	}

	centData := make([]byte, nClusters*Dimensions*4)

	for i, c := range centroids {
		base := i * Dimensions * 4

		for j, v := range c {
			binary.LittleEndian.PutUint32(centData[base+j*4:], math.Float32bits(v))
		}
	}

	if err := write("centroids.bin", centData); err != nil {
		return err
	}

	boundsData := make([]byte, (nClusters+1)*4)

	for i, b := range bounds {
		binary.LittleEndian.PutUint32(boundsData[i*4:], b)
	}

	if err := write("cluster_bounds.bin", boundsData); err != nil {
		return err
	}

	hdr := make([]byte, 8)

	binary.LittleEndian.PutUint32(hdr[0:4], uint32(nVectors))
	binary.LittleEndian.PutUint32(hdr[4:8], uint32(nClusters))

	if err := write("ivf.bin", hdr); err != nil {
		return err
	}

	log.Printf("IVF written to %s", outputPath)

	return nil
}

func kmeansPP(data [][]float32, k int) [][]float32 {
	n := len(data)

	centroids := make([][]float32, k)

	first := rand.IntN(n)

	centroids[0] = make([]float32, Dimensions)
	copy(centroids[0], data[first])

	dists := make([]float64, n)

	for c := 1; c < k; c++ {
		var total float64

		for i, vec := range data {
			_, bestDist := nearestCentroidDist(centroids[:c], vec)
			d := float64(bestDist * bestDist)
			dists[i] = d
			total += d
		}

		t := rand.Float64() * total

		var cum float64
		chosen := 0

		for i, d := range dists {
			cum += d

			if cum >= t {
				chosen = i
				break
			}
		}

		centroids[c] = make([]float32, Dimensions)
		copy(centroids[c], data[chosen])
	}

	return centroids
}
