// Package fs implements abstractions for reading files.
package fs

import (
	"errors"
	"fmt"

	"github.com/dop251/goja"

	"go.k6.io/k6/js/common"
	"go.k6.io/k6/js/modules"
	"go.k6.io/k6/lib/fsext"
)

type (
	// RootModule is the global module instance that will create create instances
	// of our module for each VU.
	RootModule struct {
		FileRegistry *fileRegistry
	}

	// ModuleInstance represents an instance of the fs module.
	ModuleInstance struct {
		vu modules.VU

		// fileRegistry holds a pointer to the global file registry.
		fileRegistry *fileRegistry
	}
)

var (
	_ modules.Module   = &RootModule{}
	_ modules.Instance = &ModuleInstance{}
)

// New returns a pointer to a new RootModule instance
func New() *RootModule {
	return &RootModule{
		FileRegistry: &fileRegistry{
			files: make(map[string]*fileData),
		},
	}
}

// NewModuleInstance implements the modules.Module interface and returns
// a new instance for each VU.
func (rm *RootModule) NewModuleInstance(vu modules.VU) modules.Instance {
	return &ModuleInstance{
		vu: vu,
		// mappedFiles: make(map[string]*mmap.MMap)
		fileRegistry: rm.FileRegistry,
	}
}

// Exports implements the modules.Instance interface and returns
// a new instance for each VU.
func (mi *ModuleInstance) Exports() modules.Exports {
	return modules.Exports{
		Named: map[string]interface{}{
			"open":             mi.Open,
			"openSync":         mi.OpenSync,
			"readFile":         mi.ReadFile,
			"readFileSync":     mi.ReadFileSync,
			"readTextFile":     mi.ReadTextFile,
			"readTextFileSync": mi.ReadTextFileSync,
			"SeekMode": map[string]SeekMode{
				"Start":   SeekModeStart,
				"Current": SeekModeCurrent,
				"End":     SeekModeEnd,
			},
		},
	}
}

// ReadFile reads a file and returns a promise that will resolve to its content.
func (mi *ModuleInstance) ReadFile(filename string) *goja.Promise {
	promise, resolve, reject := makeHandledPromise(mi.vu)

	go func() {
		fileContent, err := mi.readFile(filename)
		if err != nil {
			reject(err)
			return
		}

		rt := mi.vu.Runtime()
		ab := rt.NewArrayBuffer(fileContent)

		resolve(rt.ToValue(ab))
	}()

	return promise
}

// ReadFileSync reads a file and returns its content.
func (mi *ModuleInstance) ReadFileSync(filename string) (goja.Value, error) {
	fileContent, err := mi.readFile(filename)
	if err != nil {
		return nil, err
	}

	rt := mi.vu.Runtime()
	ab := rt.NewArrayBuffer(fileContent)

	return rt.ToValue(ab), nil
}

// ReadTextFile reads a text file and returns a promise that will resolve to its content.
func (mi *ModuleInstance) ReadTextFile(filename string) *goja.Promise {
	promise, resolve, reject := makeHandledPromise(mi.vu)

	go func() {
		fileContent, err := mi.readFile(filename)
		if err != nil {
			reject(err)
			return
		}

		resolve(string(fileContent))
	}()

	return promise
}

// ReadTextFileSync reads a text file and returns its content.
func (mi *ModuleInstance) ReadTextFileSync(filename string) (string, error) {
	fileContent, err := mi.readFile(filename)
	if err != nil {
		return "", err
	}

	return string(fileContent), nil
}

// Open opens a file and returns a promise that will resolve to a File object.
func (mi *ModuleInstance) Open(filename string) *goja.Promise {
	rt := mi.vu.Runtime()

	// if mi.vu.State() != nil {
	// 	common.Throw(rt, errors.New("open() can't be used in init context"))
	// }

	if filename == "" {
		common.Throw(rt, errors.New("open() can't be used with an empty filename"))
	}

	initEnv := mi.vu.InitEnv()
	filename = initEnv.GetAbsFilePath(filename)

	fs := initEnv.FileSystems["file"]
	if isDir, err := fsext.IsDir(fs, filename); err != nil {
		common.Throw(rt, err)
	} else if isDir {
		common.Throw(rt, fmt.Errorf("open() can't open a directory; reason: %q is a directory", filename))
	}

	promise, resolve, reject := makeHandledPromise(mi.vu)
	go func() {
		fs, ok := mi.vu.InitEnv().FileSystems["file"]
		if !ok {
			reject(errors.New("no file system configured"))
			return
		}

		f, err := mi.fileRegistry.open(filename, fs)
		if err != nil {
			reject(err)
			return
		}

		resolve(f)
	}()

	return promise
}

// OpenSync opens a file and returns a File object.
func (mi *ModuleInstance) OpenSync(filename string) goja.Value {
	rt := mi.vu.Runtime()

	if mi.vu.State() != nil {
		common.Throw(rt, errors.New("readTextFileSync() can't be used in init context"))
	}

	if filename == "" {
		common.Throw(rt, errors.New("open() can't be used with an empty filename"))
	}

	initEnv := mi.vu.InitEnv()
	filename = mi.vu.InitEnv().GetAbsFilePath(filename)

	fs := initEnv.FileSystems["file"]
	if isDir, err := fsext.IsDir(fs, filename); err != nil {
		common.Throw(rt, err)
	} else if isDir {
		common.Throw(rt, fmt.Errorf("open() can't open a directory; reason: %q is a directory", filename))
	}

	f, err := mi.fileRegistry.open(filename, fs)
	if err != nil {
		common.Throw(rt, err)
	}

	return rt.ToValue(File{vu: mi.vu, data: f})
}

// readFile reads a file and returns its content.
func (mi *ModuleInstance) readFile(filename string) ([]byte, error) {
	if mi.vu.State() != nil {
		common.Throw(mi.vu.Runtime(), errors.New("readTextFileSync() can't be used in init context"))
	}

	if filename == "" {
		common.Throw(mi.vu.Runtime(), errors.New("open() can't be used with an empty filename"))
	}

	initEnv := mi.vu.InitEnv()
	filename = initEnv.GetAbsFilePath(filename)

	fs := initEnv.FileSystems["file"]
	if isDir, err := fsext.IsDir(fs, filename); err != nil {
		return nil, err
	} else if isDir {
		return nil, fmt.Errorf("open() can't open a directory; reason: %q is a directory", filename)
	}

	data, err := fsext.ReadFile(fs, filename)
	if err != nil {
		if errors.Is(err, fsext.ErrPathNeverRequestedBefore) {
			// loading different files per VU is not supported, so all files should are going
			// to be used inside the scenario should be opened during the init step (without any conditions)
			err = fmt.Errorf(
				"open() can't be used with files that weren't previously opened during initialization (__VU==0), path: %q",
				filename,
			)
		}

		return nil, err
	}

	return data, nil
}
