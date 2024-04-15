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
		"ReadableStream":              mi.NewReadableStream,
		"CountQueuingStrategy":        mi.NewCountQueuingStrategy,
		"ReadableStreamDefaultReader": mi.NewReadableStreamDefaultReader,
	}}
}

// NewReadableStream is the constructor for the ReadableStream object.
func (mi *ModuleInstance) NewReadableStream(call goja.ConstructorCall) *goja.Object {
	runtime := mi.vu.Runtime()
	var err error

	// 1. If underlyingSource is missing, set it to null.
	var underlyingSource *goja.Object = nil

	var (
		strategy             *goja.Object
		underlyingSourceDict UnderlyingSource
	)

	// We look for the queuing strategy first, and validate it before
	// the underlying source, in order to pass the Web Platform Tests
	// constructor tests.
	strategy = mi.initializeStrategy(call)

	// 2. Let underlyingSourceDict be underlyingSource, converted to an IDL value of type UnderlyingSource.
	if len(call.Arguments) > 0 && !goja.IsUndefined(call.Arguments[0]) {
		// We first assert that it is an object (requirement)
		if !isObject(call.Arguments[0]) {
			throw(runtime, newError(TypeError, "underlyingSource must be an object"))
		}

		// Then we try to convert it to an UnderlyingSource
		underlyingSource = call.Arguments[0].ToObject(runtime)
		underlyingSourceDict, err = NewUnderlyingSourceFromObject(runtime, underlyingSource)
		if err != nil {
			throw(runtime, err)
		}
	}

	// 3. Perform ! InitializeReadableStream(this).
	stream := &ReadableStream{
		runtime: mi.vu.Runtime(),
		vu:      mi.vu,
	}
	stream.initialize()

	// 4. If underlyingSourceDict["type"] is "bytes":
	if underlyingSourceDict.Type == "bytes" {
		// 4.1. If strategy["size"] exists, throw a RangeError exception.
		if strategy.Get("size") != nil {
			common.Throw(runtime, newError(RangeError, "size function must not be set for byte streams"))
		}

		// 4.2. Let highWaterMark be ? ExtractHighWaterMark(strategy, 0).
		highWaterMark := extractHighWaterMark(runtime, strategy, 0)

		// 4.3. Perform ? SetUpReadableByteStreamControllerFromUnderlyingSource(this, underlyingSource, underlyingSourceDict, highWaterMark).
		stream.setupReadableByteStreamControllerFromUnderlyingSource(underlyingSource, underlyingSourceDict, highWaterMark)
	} else { // 5. Otherwise,
		// 5.1. Assert: underlyingSourceDict["type"] does not exist.
		if underlyingSourceDict.Type != "" {
			common.Throw(runtime, newError(AssertionError, "type must not be set for non-byte streams"))
		}

		// 5.2. Let sizeAlgorithm be ! ExtractSizeAlgorithm(strategy).
		sizeAlgorithm := extractSizeAlgorithm(runtime, strategy)

		// 5.3. Let highWaterMark be ? ExtractHighWaterMark(strategy, 1).
		highWaterMark := extractHighWaterMark(runtime, strategy, 1)

		// 5.4. Perform ? SetUpReadableStreamDefaultControllerFromUnderlyingSource(this, underlyingSource, underlyingSourceDict, highWaterMark, sizeAlgorithm).
		stream.setupReadableStreamDefaultControllerFromUnderlyingSource(underlyingSource, underlyingSourceDict, highWaterMark, sizeAlgorithm)
	}

	return runtime.ToValue(stream).ToObject(runtime)
}
func defaultSizeFunc(_ goja.Value) (float64, error) { return 1.0, nil }

func (mi *ModuleInstance) initializeStrategy(call goja.ConstructorCall) *goja.Object {
	runtime := mi.vu.Runtime()

	// Either if the strategy is not provided or if it doesn't have a 'highWaterMark',
	// we need to set its default value (highWaterMark=1).
	// https://streams.spec.whatwg.org/#rs-prototype
	strArg := runtime.NewObject()
	if len(call.Arguments) > 1 && !common.IsNullish(call.Arguments[1]) {
		strArg = call.Arguments[1].ToObject(runtime)
	}
	if common.IsNullish(strArg.Get("highWaterMark")) {
		if err := strArg.Set("highWaterMark", runtime.ToValue(1)); err != nil {
			common.Throw(runtime, newError(RuntimeError, err.Error()))
		}
	}

	// If the stream type is 'bytes', we don't want the size function.
	// Except, when it is manually specified.
	size := runtime.ToValue(defaultSizeFunc)
	if len(call.Arguments) > 0 && !common.IsNullish(call.Arguments[0]) {
		srcArg := call.Arguments[0].ToObject(runtime)
		if !common.IsNullish(srcArg.Get("type")) && srcArg.Get("type").String() == ReadableStreamTypeBytes {
			size = nil
		}
	}
	if strArg.Get("size") != nil {
		size = strArg.Get("size")
	}

	strCall := goja.ConstructorCall{Arguments: []goja.Value{strArg}}
	return mi.newCountQueuingStrategy(runtime, strCall, size)
}

// NewCountQueuingStrategy is the constructor for the [CountQueuingStrategy] object.
//
// [CountQueuingStrategy]: https://streams.spec.whatwg.org/#cqs-class
func (mi *ModuleInstance) NewCountQueuingStrategy(call goja.ConstructorCall) *goja.Object {
	rt := mi.vu.Runtime()
	// By default, the CountQueuingStrategy has a pre-defined 'size' property.
	// It cannot be overwritten by the user.
	return mi.newCountQueuingStrategy(rt, call, rt.ToValue(defaultSizeFunc))
}

// newCountQueuingStrategy is the underlying constructor for the [CountQueuingStrategy] object.
//
// It allows to create a CountQueuingStrategy with or without the 'size' property,
// depending on how the containing ReadableStream is initialized.
func (mi *ModuleInstance) newCountQueuingStrategy(rt *goja.Runtime, call goja.ConstructorCall, size goja.Value) *goja.Object {
	obj := rt.NewObject()

	if len(call.Arguments) != 1 {
		common.Throw(rt, newError(TypeError, "CountQueuingStrategy takes a single argument"))
	}

	if !isObject(call.Argument(0)) {
		common.Throw(rt, newError(TypeError, "CountQueuingStrategy argument must be an object"))
	}

	argObj := call.Argument(0).ToObject(rt)
	if common.IsNullish(argObj.Get("highWaterMark")) {
		common.Throw(rt, newError(TypeError, "CountQueuingStrategy argument must have 'highWaterMark' property"))
	}

	highWaterMark := argObj.Get("highWaterMark")
	if err := setReadOnlyPropertyOf(obj, "highWaterMark", highWaterMark); err != nil {
		common.Throw(rt, newError(TypeError, err.Error()))
	}

	if !common.IsNullish(size) {
		if err := setReadOnlyPropertyOf(obj, "size", size); err != nil {
			common.Throw(rt, newError(TypeError, err.Error()))
		}
	}

	return obj
}

// extractHighWaterMark returns the high watermark for the given queuing strategy.
//
// It implements the [ExtractHighWaterMark] algorithm.
//
// [ExtractHighWaterMark]: https://streams.spec.whatwg.org/#validate-and-normalize-high-water-mark
func extractHighWaterMark(rt *goja.Runtime, strategy *goja.Object, defaultHWM float64) float64 {
	// 1. If strategy["highWaterMark"] does not exist, return defaultHWM.
	if common.IsNullish(strategy.Get("highWaterMark")) {
		return defaultHWM
	}

	// 2. Let highWaterMark be strategy["highWaterMark"].
	highWaterMark := strategy.Get("highWaterMark")

	// 3. If highWaterMark is NaN or highWaterMark < 0, throw a RangeError exception.
	if goja.IsNaN(strategy.Get("highWaterMark")) || !isNumber(strategy.Get("highWaterMark")) || !isNonNegativeNumber(strategy.Get("highWaterMark")) {
		throw(rt, newError(RangeError, "highWaterMark must be a non-negative number"))
	}

	// 4. Return highWaterMark.
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

// NewReadableStreamDefaultReader is the constructor for the [ReadableStreamDefaultReader] object.
//
// [ReadableStreamDefaultReader]: https://streams.spec.whatwg.org/#readablestreamdefaultreader
func (mi *ModuleInstance) NewReadableStreamDefaultReader(call goja.ConstructorCall) *goja.Object {
	rt := mi.vu.Runtime()

	if len(call.Arguments) != 1 {
		throw(rt, newError(TypeError, "ReadableStreamDefaultReader takes a single argument"))
	}

	stream, ok := call.Argument(0).Export().(*ReadableStream)
	if !ok {
		throw(rt, newError(TypeError, "ReadableStreamDefaultReader argument must be a ReadableStream"))
	}

	// 1. Perform ? SetUpReadableStreamDefaultReader(this, stream).
	reader := &ReadableStreamDefaultReader{}
	reader.setup(stream)

	return rt.ToValue(reader).ToObject(rt)
}
