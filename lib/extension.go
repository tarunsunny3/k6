package lib

import (
	"fmt"
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
	fmt.Printf(">>> reflected type kind: %#+v\n", t.Kind().String())
	val := reflect.ValueOf(mod)
	fmt.Printf(">>> package path before: %#+v\n", runtime.FuncForPC(val.Pointer()).Name())
	p := t.PkgPath()
	if p == "" {
		if t.Kind() != reflect.Ptr {
			return ""
		}
		if t.Elem() != nil {
			p = t.Elem().PkgPath()
		}
	}
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	for _, dep := range buildInfo.Deps {
		packagePath := strings.TrimSpace(dep.Path)
		fmt.Printf(">>> packagePath: %s\n", packagePath)
		if strings.HasPrefix(p, packagePath) {
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
