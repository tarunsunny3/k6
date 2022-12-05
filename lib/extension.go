package lib

import (
	"reflect"
	"runtime"
	"runtime/debug"
	"strings"
)

type Module struct {
	Mod     interface{}
	Version string
}

func GetModuleVersion(mod interface{}) string {
	t := reflect.TypeOf(mod)
	path := t.PkgPath()

	if path == "" {
		switch t.Kind() {
		case reflect.Ptr:
			if t.Elem() != nil {
				path = t.Elem().PkgPath()
			}
		case reflect.Func:
			path = runtime.FuncForPC(reflect.ValueOf(mod).Pointer()).Name()
		default:
			return ""
		}
	}

	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}

	// fmt.Printf(">>> mod path: %s\n", path)
	for _, dep := range buildInfo.Deps {
		depPath := strings.TrimSpace(dep.Path)
		// fmt.Printf(">>> dep path: %s\n", depPath)
		if strings.HasPrefix(path, depPath) {
			if dep.Replace != nil {
				return dep.Replace.Version
			}
			return dep.Version
			// if _, ok := moduleVersions[packagePath]; ok {
			// 	return
			// }
			// if dep.Replace != nil {
			// 	moduleVersions[packagePath] = dep.Replace.Version
			// } else {
			// 	moduleVersions[packagePath] = dep.Version
			// }
			// break
		}
	}

	return ""
}
