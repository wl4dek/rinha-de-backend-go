package ann

import (
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sort"
	"unsafe"

	"github.com/chewxy/math32"

	"github.com/coder/hnsw"
	"golang.org/x/sys/unix"
)

const Dimensions = 14

type IVFIndex struct {
	centroids    *hnsw.Graph[int]
	centroidsRaw [][]float32
	labels       []byte
	bounds       []uint32
	vectorData   []byte
	nVectors     int
	nClusters    int
}

func (idx *IVFIndex) vectorAt(pos int) []float32 {
	offset := pos * Dimensions * 4
	return unsafe.Slice((*float32)(unsafe.Pointer(&idx.vectorData[offset])), Dimensions)
}

func (idx *IVFIndex) labelFor(pos int) string {
	if idx.labels[pos] == 1 {
		return "fraud"
	}
	return "legit"
}

func cosineDist(a, b []float32) float32 {
	var dot, nA, nB float32
	for i := range a {
		dot += float32(a[i]) * float32(b[i])
		nA += float32(a[i]) * float32(a[i])
		nB += float32(b[i]) * float32(b[i])
	}
	if nA == 0 || nB == 0 {
		return 1
	}
	return 1 - dot/(math32.Sqrt(nA)*math32.Sqrt(nB))
}

func (idx *IVFIndex) Search(query []float32, k int) ([]SearchResult, error) {
	if idx.nClusters == 0 {
		return nil, nil
	}

	nearest := idx.centroids.Search(query, 1)
	if len(nearest) == 0 {
		return nil, nil
	}
	cid := nearest[0].Key

	start := int(idx.bounds[cid])
	end := int(idx.bounds[cid+1])

	type candidate struct {
		pos  int
		dist float32
	}
	candidates := make([]candidate, 0, end-start)

	for pos := start; pos < end; pos++ {
		dist := cosineDist(idx.vectorAt(pos), query)
		candidates = append(candidates, candidate{pos, dist})
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].dist < candidates[j].dist
	})
	if len(candidates) > k {
		candidates = candidates[:k]
	}

	results := make([]SearchResult, len(candidates))
	for i, c := range candidates {
		results[i] = SearchResult{
			Label: idx.labelFor(c.pos),
			Dist:  c.dist,
		}
	}
	return results, nil
}

var pageSize = os.Getpagesize()

func roundUp(v, align int) int {
	return (v + align - 1) & ^(align - 1)
}

func mmapFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := int(fi.Size())
	if size == 0 {
		return nil, fmt.Errorf("%s is empty", path)
	}

	data, err := unix.Mmap(int(f.Fd()), 0, size, unix.PROT_READ, unix.MAP_SHARED)
	if err != nil {
		return nil, fmt.Errorf("mmap %s: %w", path, err)
	}

	if err := unix.Madvise(data, unix.MADV_RANDOM); err != nil {
		log.Printf("madvise random: %v", err)
	}

	return data, nil
}

func readUint32File(path string) ([]uint32, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data)%4 != 0 {
		return nil, fmt.Errorf("length %d not aligned to 4 in %s", len(data), path)
	}
	out := make([]uint32, len(data)/4)
	for i := range out {
		out[i] = binary.LittleEndian.Uint32(data[i*4:])
	}
	return out, nil
}

func readFloat32Vectors(path string, nVectors int) ([][]float32, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	expected := nVectors * Dimensions * 4
	if len(data) != expected {
		return nil, fmt.Errorf("expected %d bytes, got %d", expected, len(data))
	}
	out := make([][]float32, nVectors)
	for i := range out {
		out[i] = unsafe.Slice((*float32)(unsafe.Pointer(&data[i*Dimensions*4])), Dimensions)
	}
	return out, nil
}

func LoadIVF(path string) (*IVFIndex, error) {
	hdr, err := os.ReadFile(filepath.Join(path, "ivf.bin"))
	if err != nil {
		return nil, fmt.Errorf("read ivf.bin: %w", err)
	}
	if len(hdr) < 8 {
		return nil, fmt.Errorf("ivf.bin too short")
	}
	nVectors := int(binary.LittleEndian.Uint32(hdr[0:4]))
	nClusters := int(binary.LittleEndian.Uint32(hdr[4:8]))
	log.Printf("Loading IVF index: %d vectors, %d clusters", nVectors, nClusters)

	labels, err := os.ReadFile(filepath.Join(path, "labels.bin"))
	if err != nil {
		return nil, fmt.Errorf("labels: %w", err)
	}
	if len(labels) != nVectors {
		return nil, fmt.Errorf("labels len %d != %d", len(labels), nVectors)
	}

	bounds, err := readUint32File(filepath.Join(path, "cluster_bounds.bin"))
	if err != nil {
		return nil, fmt.Errorf("cluster_bounds: %w", err)
	}
	if len(bounds) != nClusters+1 {
		return nil, fmt.Errorf("cluster_bounds len %d != %d", len(bounds), nClusters+1)
	}

	vectorData, err := mmapFile(filepath.Join(path, "vectors.bin"))
	if err != nil {
		return nil, fmt.Errorf("vectors mmap: %w", err)
	}

	centroidsRaw, err := readFloat32Vectors(filepath.Join(path, "centroids.bin"), nClusters)
	if err != nil {
		return nil, fmt.Errorf("centroids: %w", err)
	}

	g := hnsw.NewGraph[int]()
	g.M = 16
	g.EfSearch = 20

	nodes := make([]hnsw.Node[int], nClusters)
	for i, vec := range centroidsRaw {
		nodes[i] = hnsw.MakeNode(i, vec)
	}
	g.Add(nodes...)

	return &IVFIndex{
		centroids:    g,
		centroidsRaw: centroidsRaw,
		labels:       labels,
		bounds:       bounds,
		vectorData:   vectorData,
		nVectors:     nVectors,
		nClusters:    nClusters,
	}, nil
}

func cosineSimilarity(a, b []float32) float32 {
	var dot, nA, nB float32
	for i := range a {
		dot += float32(a[i]) * float32(b[i])
		nA += float32(a[i]) * float32(a[i])
		nB += float32(b[i]) * float32(b[i])
	}
	if nA == 0 || nB == 0 {
		return 0
	}
	return dot / (math32.Sqrt(nA) * math32.Sqrt(nB))
}

func nearestCentroid(centroids [][]float32, vec []float32) int {
	best := 0
	bestSim := float32(-2)
	for i, c := range centroids {
		sim := cosineSimilarity(c, vec)
		if sim > bestSim {
			bestSim = sim
			best = i
		}
	}
	return best
}

func BuildIVF(refs []Reference, nClusters int, outputPath string) error {
	nVectors := len(refs)
	if nVectors == 0 {
		return fmt.Errorf("no references")
	}
	sampleSize := min(50000, nVectors)
	log.Printf("Building IVF: %d vectors, %d clusters, sample %d", nVectors, nClusters, sampleSize)

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
			if counts[i] > 0 {
				for j := range newCentroids[i] {
					newCentroids[i][j] /= float32(counts[i])
				}
			} else {
				copy(newCentroids[i], centroids[i])
			}
		}

		var shift float64
		for i := range newCentroids {
			for j := range newCentroids[i] {
				d := float64(newCentroids[i][j] - centroids[i][j])
				shift += d * d
			}
		}
		copy(centroids, newCentroids)
		if shift < 1e-10 {
			log.Printf("K-means converged at iteration %d", iter)
			break
		}
	}
	log.Println("K-means done")

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

	log.Printf("Writing vectors.bin (%d MB)", nVectors*Dimensions*4/1024/1024)
	vecData := make([]byte, nVectors*Dimensions*4)
	for dstIdx, srcIdx := range order {
		for j, v := range refs[srcIdx].Vector {
			binary.LittleEndian.PutUint32(vecData[(dstIdx*Dimensions+j)*4:], math.Float32bits(v))
		}
	}
	if err := write("vectors.bin", vecData); err != nil {
		return err
	}

	log.Printf("Writing labels.bin (%d MB)", nVectors/1024/1024)
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
		for j, v := range c {
			binary.LittleEndian.PutUint32(centData[(i*Dimensions+j)*4:], math.Float32bits(v))
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

	log.Printf("IVF index written to %s", outputPath)
	return nil
}

func addVec(to, from []float32) {
	for i := range to {
		to[i] += from[i]
	}
}

func kmeansPP(data [][]float32, k int) [][]float32 {
	n := len(data)
	centroids := make([][]float32, k)

	first := rand.IntN(n)
	centroids[0] = make([]float32, Dimensions)
	copy(centroids[0], data[first])

	dists := make([]float32, n)
	for c := 1; c < k; c++ {
		var total float32
		for i, vec := range data {
			_, bestDist := nearestCentroidDist(centroids[:c], vec)
			dists[i] = bestDist * bestDist
			total += dists[i]
		}
		if total == 0 {
			centroids[c] = make([]float32, Dimensions)
			copy(centroids[c], data[rand.IntN(n)])
			continue
		}
		t := rand.Float32() * total
		var cum float32
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

func nearestCentroidDist(centroids [][]float32, vec []float32) (int, float32) {
	best := 0
	bestDist := float32(2.0)
	for i, c := range centroids {
		sim := cosineSimilarity(c, vec)
		d := 1 - float32(sim)
		if d < bestDist {
			bestDist = d
			best = i
		}
	}
	return best, bestDist
}
