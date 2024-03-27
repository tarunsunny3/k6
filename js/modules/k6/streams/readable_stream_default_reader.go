package streams

import (
	"github.com/dop251/goja"
	"go.k6.io/k6/js/common"
	"go.k6.io/k6/js/promises"
)

// ReadableStreamDefaultReader represents a default reader designed to be vended by a [ReadableStream].
type ReadableStreamDefaultReader struct {
	BaseReadableStreamReader

	// readRequests holds a list of read requests, used when a consumer requests
	// chunks sooner than they are available.
	readRequests []ReadRequest
}

// NewReadableStreamDefaultReaderObject creates a new goja.Object from a [ReadableStreamDefaultReader] instance.
func NewReadableStreamDefaultReaderObject(reader *ReadableStreamDefaultReader) (*goja.Object, error) {
	rt := reader.stream.runtime
	obj := rt.NewObject()

	err := obj.DefineAccessorProperty("closed", rt.ToValue(func() *goja.Promise {
		p, _, _ := reader.GetClosed()
		return p
	}), nil, goja.FLAG_FALSE, goja.FLAG_TRUE)
	if err != nil {
		return nil, err
	}

	if err := setReadOnlyPropertyOf(obj, "cancel", rt.ToValue(reader.Cancel)); err != nil {
		return nil, err
	}

	// Exposing the properties of the [ReadableStreamDefaultReader] interface
	if err := setReadOnlyPropertyOf(obj, "read", rt.ToValue(reader.Read)); err != nil {
		return nil, err
	}

	if err := setReadOnlyPropertyOf(obj, "releaseLock", rt.ToValue(reader.ReleaseLock)); err != nil {
		return nil, err
	}

	return obj, nil
}

// Ensure the ReadableStreamReader interface is implemented correctly
var _ ReadableStreamReader = &ReadableStreamDefaultReader{}

// Read returns a [goja.Promise] providing access to the next chunk in the stream's internal queue.
func (reader *ReadableStreamDefaultReader) Read() *goja.Promise {
	stream := reader.GetStream()

	// 1. If this.[[stream]] is undefined, return a promise rejected with a TypeError exception.
	if stream == nil {
		return newRejectedPromise(stream.vu, newError(TypeError, "stream is undefined"))
	}

	// 2. Let promise be a new promise.
	promise, resolve, reject := promises.New(stream.vu)

	// 3. Let readRequest be a new read request with the following items:
	readRequest := ReadRequest{
		chunkSteps: func(chunk any) {
			go func() {
				// Resolve promise with «[ "value" → chunk, "done" → false ]».
				resolve(map[string]any{"value": chunk, "done": false})
			}()
		},
		closeSteps: func() {
			go func() {
				// Resolve promise with «[ "value" → undefined, "done" → true ]».
				resolve(map[string]any{"value": goja.Undefined(), "done": true})
			}()
		},
		errorSteps: func(e any) {
			go func() {
				// Reject promise with e.
				reject(e)
			}()
		},
	}

	// 4. Perform ! ReadableStreamDefaultReaderRead(this, readRequest).
	go func() {
		reader.read(readRequest)
	}()

	// 5. Return promise.
	return promise
}

// Closed returns a [goja.Promise] that fulfills when the stream closes, or
// rejects if the stream throws an error or the reader's lock is released.
//
// This property enables you to write code that responds to an end to the streaming process.
func (reader *ReadableStreamDefaultReader) Closed() *goja.Promise {
	// FIXME: should be exposed as a property instead of a method
	// Implement logic to return a promise that fulfills or rejects based on the reader's state
	// The promise should fulfill when the reader is closed and reject if the reader is errored
	return nil
}

// Cancel returns a [goja.Promise] that resolves when the stream is canceled.
//
// Calling this method signals a loss of interest in the stream by a consumer. The
// supplied reason argument will be given to the underlying source, which may or
// may not use it.
//
// The `reason` argument is optional, and should hold A human-readable reason for
// the cancellation. This value may or may not be used.
// FIXME: implement according to specification.
func (reader *ReadableStreamDefaultReader) Cancel(_ goja.Value) *goja.Promise {
	// Implement logic to return a promise that fulfills or rejects based on the reader's state
	// The promise should fulfill when the reader is closed and reject if the reader is errored
	return nil
}

// ReadResult is the result of a read operation
//
// It contains the value read from the stream and a boolean indicating whether or not the stream is done.
// An undefined value indicates that the stream has been closed.
type ReadResult struct {
	Value goja.Value
	Done  bool
}

// ReleaseLock releases the reader's lock on the stream.
//
// If the associated stream is errored when the lock is released, the
// reader will appear errored in that same way subsequently; otherwise, the
// reader will appear closed.
func (reader *ReadableStreamDefaultReader) ReleaseLock() {
	// Implement the logic to release the lock on the stream
	// This might involve changing the state of the stream and handling any queued read requests
}

// setup implements the [SetUpReadableStreamDefaultReader] algorithm.
//
// [SetUpReadableStreamDefaultReader]: https://streams.spec.whatwg.org/#set-up-readable-stream-default-reader
func (reader *ReadableStreamDefaultReader) setup(stream *ReadableStream) {
	// 1. If ! IsReadableStreamLocked(stream) is true, throw a TypeError exception.
	if stream.isLocked() {
		common.Throw(reader.GetStream().vu.Runtime(), newError(TypeError, "stream is locked"))
	}

	// 2. Perform ! ReadableStreamReaderGenericInitialize(reader, stream).
	ReadableStreamReaderGenericInitialize(reader, stream)

	// 3. Set reader.[[readRequests]] to a new empty list.
	reader.readRequests = []ReadRequest{}
}

// Implements the [specification]'s ReadableStreamDefaultReaderErrorReadRequests algorithm.
//
// [specification]: https://streams.spec.whatwg.org/#abstract-opdef-readablestreamdefaultreadererrorreadrequests
func (reader *ReadableStreamDefaultReader) errorReadRequests(e any) {
	// 1. Let readRequests be reader.[[readRequests]].
	readRequests := reader.readRequests

	// 2. Set reader.[[readRequests]] to a new empty list.
	reader.readRequests = []ReadRequest{}

	// 3. For each readRequest of readRequests,
	for _, request := range readRequests {
		// 3.1. Perform readRequest’s error steps, given e.
		request.errorSteps(e)
	}
}

// read implements the [ReadableStreamDefaultReaderRead] algorithm.
//
// [ReadableStreamDefaultReaderRead]: https://streams.spec.whatwg.org/#readable-stream-default-reader-read
func (reader *ReadableStreamDefaultReader) read(readRequest ReadRequest) {
	// 1. Let stream be reader.[[stream]].
	stream := reader.GetStream()

	// 2. Assert: stream is not undefined.
	if stream == nil {
		common.Throw(stream.vu.Runtime(), newError(AssertionError, "stream is undefined"))
	}

	// 3. Set stream.[[disturbed]] to true.
	stream.disturbed = true

	switch stream.state {
	case ReadableStreamStateClosed:
		// 4. If stream.[[state]] is "closed", perform readRequest’s close steps.
		readRequest.closeSteps()
	case ReadableStreamStateErrored:
		// 5. Otherwise, if stream.[[state]] is "errored", perform readRequest’s error steps given stream.[[storedError]].
		readRequest.errorSteps(stream.storedError)
	default:
		// 6. Otherwise,
		// 6.1. Assert: stream.[[state]] is "readable".
		if stream.state != ReadableStreamStateReadable {
			common.Throw(stream.vu.Runtime(), newError(AssertionError, "stream.state is not readable"))
		}

		// 6.2. Perform ! stream.[[controller]].[[PullSteps]](readRequest).
		stream.controller.pullSteps(readRequest)
	}
}
