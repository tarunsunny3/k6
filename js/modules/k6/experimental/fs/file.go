package fs

import (
	"errors"

	"github.com/dop251/goja"
	"go.k6.io/k6/js/common"
	"go.k6.io/k6/js/modules"
)

// File represents a file opened by the fs module.
type File struct {
	vu modules.VU

	// name holds the name of the file.
	name string

	// data holds a pointer to the file bytes.
	data *fileData

	// offset holds the current offset within the file.
	offset int
}

// Read the file into an array buffer.
//
// Resolves to either the number of bytes read during the operation or
// `null` if the end of the file has been reached.
func (f *File) Read(p goja.Value) *goja.Promise {
	promise, resolve, reject := makeHandledPromise(f.vu)

	buffer, err := exportArrayBuffer(f.vu.Runtime(), p)
	if err != nil {
		reject(err)
		return promise
	}

	go func() {
		n, err := f.readImpl(buffer)
		if err != nil {
			if errors.Is(err, ErrEOF) {
				resolve(goja.Null())
				return
			}

			reject(err)
			return
		}

		resolve(n)
	}()

	return promise
}

// ReadSync synchronously read the file into an array buffer.
//
// Returns either the number of bytes read during the operation or
// `null` if the end of the file has been reached.
func (f *File) ReadSync(p goja.Value) goja.Value {
	buffer, err := exportArrayBuffer(f.vu.Runtime(), p)
	if err != nil {
		common.Throw(f.vu.Runtime(), err)
	}

	n, err := f.readImpl(buffer)
	if err != nil {
		if errors.Is(err, ErrEOF) {
			return goja.Null()
		}

		common.Throw(f.vu.Runtime(), err)
	}

	return f.vu.Runtime().ToValue(n)
}

// ReadAll reads the entire file into an array buffer.
func (f *File) ReadAll() *goja.Promise {
	rt := f.vu.Runtime()

	promise, resolve, _ := makeHandledPromise(f.vu)

	go func() {
		resolve(rt.ToValue(rt.NewArrayBuffer(f.data.bytes)))
	}()

	return promise
}

// ReadAllSync synchronously reads the entire file into an array buffer.
func (f *File) ReadAllSync() goja.ArrayBuffer {
	return f.vu.Runtime().NewArrayBuffer(f.data.bytes)
}

// ErrEOF is returned when the end of the file has been reached.
var ErrEOF = errors.New("EOF")

func (f *File) readImpl(buffer []byte) (int, error) {
	start := f.offset
	if start == len(f.data.bytes) {
		return 0, ErrEOF
	}

	end := f.offset + len(buffer)
	if end > len(f.data.bytes) {
		end = len(f.data.bytes)
	}

	n := copy(buffer, f.data.bytes[start:end])

	f.offset += n

	return n, nil
}

// Seek seeks to the given `offset` under mode given by `whence`.
// The call resolves to the new position within the file (bytes from the start).
func (f *File) Seek(offset int64, whence SeekMode) *goja.Promise {
	promise, resolve, _ := makeHandledPromise(f.vu)

	go func() {
		resolve(f.seekImpl(offset, whence))
	}()

	return promise
}

// SeekSync synchronously seeks to the given `offset` under mode given by `whence`.
//
// Returns the new position within the file (bytes from the start).
func (f *File) SeekSync(offset int64, whence SeekMode) goja.Value {
	return f.vu.Runtime().ToValue(f.seekImpl(offset, whence))
}

// SeekMode represents the mode to use when seeking a file.
type SeekMode = int

const (
	// SeekModeStart seeks relative to the start of the file.
	SeekModeStart SeekMode = 0

	// SeekModeCurrent seeks relative to the current position.
	SeekModeCurrent SeekMode = 1

	// SeekModeEnd seeks relative to the end of the file. When using this mode,
	// the seek operation will move backwards from the end of the file.
	SeekModeEnd SeekMode = 2
)

func (f *File) seekImpl(offset int64, whence SeekMode) int64 {
	switch whence {
	case SeekModeStart:
		f.offset = int(offset)
	case SeekModeCurrent:
		f.offset += int(offset)
	case SeekModeEnd:
		f.offset = len(f.data.bytes) - int(offset)
	}

	if f.offset < 0 {
		f.offset = 0
	}

	if f.offset > len(f.data.bytes) {
		f.offset = len(f.data.bytes)
	}

	return int64(f.offset)
}

// FileInfo represents a file's information.
type FileInfo struct {
	Name string `json:"name"`
	Size int    `json:"size"`
}

// Stat returns a promise that will resolve to a FileInfo object.
func (f *File) Stat() *goja.Promise {
	promise, resolve, _ := makeHandledPromise(f.vu)

	go func() {
		resolve(FileInfo{
			Name: f.name,
			Size: len(f.data.bytes),
		})
	}()

	return promise
}

// StatSync returns a FileInfo object.
func (f *File) StatSync() FileInfo {
	return FileInfo{
		Name: f.name,
		Size: len(f.data.bytes),
	}
}
