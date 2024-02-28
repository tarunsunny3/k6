// Package streams provides support for the Web Streams API.
package streams

import (
	"github.com/dop251/goja"
	"go.k6.io/k6/js/common"
	"go.k6.io/k6/js/modules"
)

// FIXME: figure out what "return a promise rejects/fulfilled with" means in practice for us:
//   - Should we execute the promise's resolve/reject functions in a go func() { ... } still?
//   - So far I've made a mix of both, but I'm not sure what's the best approach, if any

// TODO: have an `Assert` helper function that throws an assertion error if the condition is not met
// TODO: Document we do not support the following:
// - static `from` constructor as it's expected to take an `asyncIterable` as input we do not support

type (
	// RootModule is the module that will be registered with the runtime.
	RootModule struct{}

	// ModuleInstance is the module instance that will be created for each VU.
	ModuleInstance struct {
		vu modules.VU
	}
)

// Ensure the interfaces are implemented correctly
var (
	_ modules.Instance = &ModuleInstance{}
	_ modules.Module   = &RootModule{}
)

// New creates a new RootModule instance.
func New() *RootModule {
	return &RootModule{}
}

// NewModuleInstance creates a new instance of the module for a specific VU.
func (rm *RootModule) NewModuleInstance(vu modules.VU) modules.Instance {
	return &ModuleInstance{
		vu: vu,
	}
}

// Exports returns the module exports, that will be available in the runtime.
func (mi *ModuleInstance) Exports() modules.Exports {
	return modules.Exports{Named: map[string]interface{}{
		"ReadableStream": mi.NewReadableStream,
	}}
}

// NewReadableStream is the constructor for the ReadableStream object.
func (mi *ModuleInstance) NewReadableStream(call goja.ConstructorCall) *goja.Object {
	runtime := mi.vu.Runtime()
	var err error

	var underlyingSource *UnderlyingSource
	var strategy *goja.Object

	// We look for the queuing strategy first, and validate it before
	// the underlying source, in order to pass the Web Platform Tests
	// constructor tests.
	if len(call.Arguments) > 1 && !common.IsNullish(call.Arguments[1]) {
		strategy = call.Arguments[1].ToObject(runtime)
	} else {
		strategy = NewCountQueuingStrategy(runtime, goja.ConstructorCall{Arguments: []goja.Value{runtime.ToValue(1)}})
	}

	if len(call.Arguments) > 0 && !common.IsNullish(call.Arguments[0]) {
		// 2.
		underlyingSource, err = NewUnderlyingSourceFromObject(runtime, call.Arguments[0].ToObject(runtime))
		if err != nil {
			common.Throw(runtime, err)
		}
	} else {
		// 1.
		underlyingSource = nil
	}

	// 3.
	stream := &ReadableStream{
		runtime: mi.vu.Runtime(),
		vu:      mi.vu,
	}
	stream.initialize()

	if underlyingSource != nil && underlyingSource.Type == "bytes" { // 4.
		// 4.1
		if strategy.Get("size") != nil {
			common.Throw(runtime, newError(RangeError, "size function must not be set for byte streams"))
		}

		// 4.2
		// highWaterMark := strategy.extractHighWaterMark(0)
		highWaterMark := extractHighWaterMark(runtime, strategy, 0)

		// 4.3
		stream.setupReadableByteStreamControllerFromUnderlyingSource(*underlyingSource, highWaterMark)
	} else { // 5.
		// 5.1
		if underlyingSource != nil && underlyingSource.Type != "" {
			common.Throw(runtime, newError(AssertionError, "type must not be set for non-byte streams"))
		}

		// 5.2
		sizeAlgorithm := extractSizeAlgorithm(runtime, strategy)

		// 5.3
		highWaterMark := extractHighWaterMark(runtime, strategy, 1)

		// 5.4
		stream.setupDefaultControllerFromUnderlyingSource(*underlyingSource, highWaterMark, sizeAlgorithm)
	}

	return runtime.ToValue(stream).ToObject(runtime)
}

// NewCountQueuingStrategy is the constructor for the CountQueuingStrategy object.
func NewCountQueuingStrategy(rt *goja.Runtime, call goja.ConstructorCall) *goja.Object {
	obj := rt.NewObject()

	if len(call.Arguments) != 1 {
		common.Throw(rt, newError(TypeError, "CountQueuingStrategy takes a single argument"))
	}

	highWaterMark := call.Argument(0)
	if err := setReadOnlyPropertyOf(obj, "highWaterMark", highWaterMark); err != nil {
		common.Throw(rt, newError(TypeError, err.Error()))
	}

	sizeFunc := func(_ goja.Value) (float64, error) { return 1.0, nil }
	if err := setReadOnlyPropertyOf(obj, "size", rt.ToValue(sizeFunc)); err != nil {
		common.Throw(rt, newError(TypeError, err.Error()))
	}

	return obj
}

// extractHighWaterMark returns the high water mark for the given queuing strategy.
//
// It implements the [ExtractHighWaterMark] algorithm.
//
// [ExtractHighWaterMark]: https://streams.spec.whatwg.org/#validate-and-normalize-high-water-mark
func extractHighWaterMark(rt *goja.Runtime, strategy *goja.Object, defaultHWM float64) float64 {
	highWaterMark := strategy.Get("highWaterMark")

	// 1.
	if common.IsNullish(highWaterMark) {
		return defaultHWM
	}

	// 2. 3.
	if goja.IsNaN(highWaterMark) || !isNumber(highWaterMark) || !isNonNegativeNumber(highWaterMark) {
		common.Throw(rt, newError(RangeError, "highWaterMark must be a non-negative number"))
	}

	return highWaterMark.ToFloat()
}

// extractSizeAlgorithm returns the size algorithm for the given queuing strategy.
//
// It implements the [ExtractSizeAlgorithm] algorithm.
//
// [ExtractSizeAlgorithm]: https://streams.spec.whatwg.org/#make-size-algorithm-from-size-function
func extractSizeAlgorithm(rt *goja.Runtime, strategy *goja.Object) SizeAlgorithm {
	var sizeFunc goja.Callable
	sizeProp := strategy.Get("size")

	if common.IsNullish(sizeProp) {
		sizeFunc, _ = goja.AssertFunction(rt.ToValue(func(_ goja.Value) (float64, error) { return 1.0, nil }))
		return sizeFunc
	}

	sizeFunc, isFunc := goja.AssertFunction(sizeProp)
	if !isFunc {
		common.Throw(rt, newError(TypeError, "size must be a function"))
	}

	return sizeFunc
}
