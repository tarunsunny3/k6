package fs

import (
	"io"
	"sync"

	"github.com/spf13/afero"
)

// fileRegistry is a registry of memory-mapped files.
type fileRegistry struct {
	files map[string]*fileData

	// mutex is used to synchronize access to the memory-mapped files.
	mutex sync.Mutex
}

// open opens a file and memory maps it.
// func (fr *fileRegistry) open(filename string) (*mmap.MMap, error) {
func (fr *fileRegistry) open(filename string, fromFs afero.Fs) (*fileData, error) {
	fr.mutex.Lock()
	defer fr.mutex.Unlock()
	if f, ok := fr.files[filename]; ok {
		return f, nil
	}

	f, err := fromFs.Open(filename)
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	fd := &fileData{
		bytes: data,
	}

	fr.files[filename] = fd

	return fd, nil
}

type fileData struct {
	bytes []byte
}
