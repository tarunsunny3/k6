package ext

import (
	"fmt"
	"reflect"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
)

type ExtensionType uint8

const (
	JSExtension ExtensionType = iota + 1
	OutputExtension
)

func (e ExtensionType) String() string {
	var s string
	switch e {
	case JSExtension:
		s = "js"
	case OutputExtension:
		s = "out"
	}
	return s
}

type Extension struct {
	Name, Path, Version string
	Type                ExtensionType
	Mod                 interface{}
}

func (e Extension) String() string {
	return fmt.Sprintf("[%s] %s %s %s", e.Type, e.Path, e.Name, e.Version)
}

//nolint:gochecknoglobals
// TODO: Make an ExtensionRegistry?
var (
	mx         sync.RWMutex
	extensions = make(map[ExtensionType]map[string]*Extension)
)

func Register(name string, typ ExtensionType, mod interface{}) {
	mx.Lock()
	defer mx.Unlock()

	exts, ok := extensions[typ]
	if !ok {
		panic(fmt.Sprintf("unsupported extension type: %T", typ))
	}

	if _, ok := exts[name]; ok {
		panic(fmt.Sprintf("extension already registered: %s", name))
	}

	path, version := getModuleInfo(mod)

	exts[name] = &Extension{
		Name:    name,
		Type:    typ,
		Mod:     mod,
		Path:    path,
		Version: version,
	}
}

func Get(typ ExtensionType) []*Extension {
	mx.RLock()
	defer mx.RUnlock()

	exts, ok := extensions[typ]
	if !ok {
		panic(fmt.Sprintf("unsupported extension type: %T", typ))
	}

	result := make([]*Extension, 0, len(exts))

	for _, ext := range exts {
		result = append(result, ext)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Path > result[j].Path && result[i].Name > result[j].Name
	})

	return result
}

func getModuleInfo(mod interface{}) (path, version string) {
	t := reflect.TypeOf(mod)
	path = t.PkgPath()

	if path == "" {
		switch t.Kind() {
		case reflect.Ptr:
			if t.Elem() != nil {
				path = t.Elem().PkgPath()
			}
		case reflect.Func:
			path = runtime.FuncForPC(reflect.ValueOf(mod).Pointer()).Name()
		default:
			return "", ""
		}
	}

	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return "", ""
	}

	for _, dep := range buildInfo.Deps {
		depPath := strings.TrimSpace(dep.Path)
		if strings.HasPrefix(path, depPath) {
			if dep.Replace != nil {
				return depPath, dep.Replace.Version
			}
			return depPath, dep.Version
		}
	}

	return "", ""
}

func init() {
	extensions[JSExtension] = make(map[string]*Extension)
	extensions[OutputExtension] = make(map[string]*Extension)
}
