package ann

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

func mmapFloat32(path string) ([]float32, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}

	size := int(fi.Size())

	data, err := unix.Mmap(int(f.Fd()), 0, size, unix.PROT_READ, unix.MAP_SHARED)
	if err != nil {
		return nil, err
	}

	if err := unix.Madvise(data, unix.MADV_SEQUENTIAL); err != nil {
		log.Printf("madvise sequential: %v", err)
	}

	floatCount := size / 4
	vecs := unsafe.Slice((*float32)(unsafe.Pointer(&data[0])), floatCount)

	return vecs, nil
}

func readUint32File(path string) ([]uint32, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
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
