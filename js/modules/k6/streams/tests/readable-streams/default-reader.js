// // Original source file: https://github.com/web-platform-tests/wpt/blob/3fd901dba4d461afda4cf9b692f8bd99fb05f4e1/streams/readable-streams/default-reader.any.js
// // META: global=window,worker,shadowrealm
// // META: script=../resources/rs-utils.js
'use strict';

test(() => {

	assert_throws_js(TypeError, () => new ReadableStreamDefaultReader('potato'));
	assert_throws_js(TypeError, () => new ReadableStreamDefaultReader({}));
	assert_throws_js(TypeError, () => new ReadableStreamDefaultReader());

}, 'ReadableStreamDefaultReader constructor should get a ReadableStream object as argument');

test(() => {

	const rsReader = new ReadableStreamDefaultReader(new ReadableStream());
	//assert_equals(rsReader.closed, rsReader.closed, 'closed should return the same promise');

}, 'ReadableStreamDefaultReader closed should always return the same promise object');

test(() => {

	const rs = new ReadableStream();
	new ReadableStreamDefaultReader(rs); // Constructing directly the first time should be fine.
	assert_throws_js(TypeError, () => new ReadableStreamDefaultReader(rs),
		'constructing directly the second time should fail');

}, 'Constructing a ReadableStreamDefaultReader directly should fail if the stream is already locked (via direct ' +
	'construction)');

test(() => {

	const rs = new ReadableStream();
	new ReadableStreamDefaultReader(rs); // Constructing directly should be fine.
	assert_throws_js(TypeError, () => rs.getReader(), 'getReader() should fail');

}, 'Getting a ReadableStreamDefaultReader via getReader should fail if the stream is already locked (via direct ' +
	'construction)');

test(() => {

	const rs = new ReadableStream();
	rs.getReader(); // getReader() should be fine.
	assert_throws_js(TypeError, () => new ReadableStreamDefaultReader(rs), 'constructing directly should fail');

}, 'Constructing a ReadableStreamDefaultReader directly should fail if the stream is already locked (via getReader)');

test(() => {

	const rs = new ReadableStream();
	rs.getReader(); // getReader() should be fine.
	assert_throws_js(TypeError, () => rs.getReader(), 'getReader() should fail');

}, 'Getting a ReadableStreamDefaultReader via getReader should fail if the stream is already locked (via getReader)');

test(() => {

	const rs = new ReadableStream({
		start(c) {
			c.close();
		}
	});

	new ReadableStreamDefaultReader(rs); // Constructing directly should not throw.

}, 'Constructing a ReadableStreamDefaultReader directly should be OK if the stream is closed');

test(() => {

	const theError = new Error('don\'t say i didn\'t warn ya');
	const rs = new ReadableStream({
		start(c) {
			c.error(theError);
		}
	});

	new ReadableStreamDefaultReader(rs); // Constructing directly should not throw.

}, 'Constructing a ReadableStreamDefaultReader directly should be OK if the stream is errored');

promise_test(() => {

	let controller;
	const rs = new ReadableStream({
		start(c) {
			controller = c;
		}
	});
	const reader = rs.getReader();

	const promise = reader.read().then(result => {
		assert_object_equals(result, {value: 'a', done: false}, 'read() should fulfill with the enqueued chunk');
	});

	controller.enqueue('a');
	return promise;

}, 'Reading from a reader for an empty stream will wait until a chunk is available');

promise_test(() => {

	let cancelCalled = false;
	const passedReason = new Error('it wasn\'t the right time, sorry');
	const rs = new ReadableStream({
		cancel(reason) {
			assert_true(rs.locked, 'the stream should still be locked');
			assert_throws_js(TypeError, () => rs.getReader(), 'should not be able to get another reader');
			assert_equals(reason, passedReason, 'the cancellation reason is passed through to the underlying source');
			cancelCalled = true;
		}
	});

	const reader = rs.getReader();
	return reader.cancel(passedReason).then(() => assert_true(cancelCalled));

}, 'cancel() on a reader does not release the reader');

promise_test(() => {

	let controller;
	const rs = new ReadableStream({
		start(c) {
			controller = c;
		}
	});

	const reader = rs.getReader();
	const promise = reader.closed;

	controller.close();
	return promise;

}, 'closed should be fulfilled after stream is closed (.closed access before acquiring)');

promise_test(t => {

	let controller;
	const rs = new ReadableStream({
		start(c) {
			controller = c;
		}
	});

	const reader1 = rs.getReader();

	reader1.releaseLock();

	const reader2 = rs.getReader();
	controller.close();

	return Promise.all([
		promise_rejects_js(t, TypeError, reader1.closed),
		reader2.closed
	]);

}, 'closed should be rejected after reader releases its lock (multiple stream locks)');

promise_test(t => {

	let controller;
	const rs = new ReadableStream({
		start(c) {
			controller = c;
		}
	});

	const reader = rs.getReader();
	const promise1 = reader.closed;

	controller.close();

	reader.releaseLock();
	const promise2 = reader.closed;

	assert_not_equals(promise1, promise2, '.closed should be replaced');
	return Promise.all([
		promise1,
		promise_rejects_js(t, TypeError, promise2, '.closed after releasing lock'),
	]);

}, 'closed is replaced when stream closes and reader releases its lock');

promise_test(t => {

	const theError = {name: 'unique error'};
	let controller;
	const rs = new ReadableStream({
		start(c) {
			controller = c;
		}
	});

	const reader = rs.getReader();
	const promise1 = reader.closed;

	controller.error(theError);

	reader.releaseLock();
	const promise2 = reader.closed;

	assert_not_equals(promise1, promise2, '.closed should be replaced');
	return Promise.all([
		promise_rejects_exactly(t, theError, promise1, '.closed before releasing lock'),
		promise_rejects_js(t, TypeError, promise2, '.closed after releasing lock')
	]);

}, 'closed is replaced when stream errors and reader releases its lock');

promise_test(() => {

	const rs = new ReadableStream({
		start(c) {
			c.enqueue('a');
			c.enqueue('b');
			c.close();
		}
	});

	const reader1 = rs.getReader();
	const promise1 = reader1.read().then(r => {
		assert_object_equals(r, {value: 'a', done: false}, 'reading the first chunk from reader1 works');
	});
	reader1.releaseLock();

	const reader2 = rs.getReader();
	const promise2 = reader2.read().then(r => {
		assert_object_equals(r, {value: 'b', done: false}, 'reading the second chunk from reader2 works');
	});
	reader2.releaseLock();

	return Promise.all([promise1, promise2]);

}, 'Multiple readers can access the stream in sequence');

promise_test(() => {
	const rs = new ReadableStream({
		start(c) {
			c.enqueue('a');
		}
	});

	const reader1 = rs.getReader();
	reader1.releaseLock();

	const reader2 = rs.getReader();

	// Should be a no-op
	reader1.releaseLock();

	return reader2.read().then(result => {
		assert_object_equals(result, {value: 'a', done: false},
			'read() should still work on reader2 even after reader1 is released');
	});

}, 'Cannot use an already-released reader to unlock a stream again');

promise_test(t => {

	const rs = new ReadableStream({
		start(c) {
			c.enqueue('a');
		},
		cancel() {
			assert_unreached('underlying source cancel should not be called');
		}
	});

	const reader = rs.getReader();
	reader.releaseLock();
	const cancelPromise = reader.cancel();

	const reader2 = rs.getReader();
	const readPromise = reader2.read().then(r => {
		assert_object_equals(r, { value: 'a', done: false }, 'a new reader should be able to read a chunk');
	});

	return Promise.all([
		promise_rejects_js(t, TypeError, cancelPromise),
		readPromise
	]);

}, 'cancel() on a released reader is a no-op and does not pass through');

// promise_test(t => {
//
// 	const promiseAsserts = [];
//
// 	let controller;
// 	const theError = { name: 'unique error' };
// 	const rs = new ReadableStream({
// 		start(c) {
// 			controller = c;
// 		}
// 	});
//
// 	const reader1 = rs.getReader();
//
// 	promiseAsserts.push(
// 		promise_rejects_exactly(t, theError, reader1.closed),
// 		promise_rejects_exactly(t, theError, reader1.read())
// 	);
//
// 	assert_throws_js(TypeError, () => rs.getReader(), 'trying to get another reader before erroring should throw');
//
// 	controller.error(theError);
//
// 	reader1.releaseLock();
//
// 	const reader2 = rs.getReader();
//
// 	promiseAsserts.push(
// 		promise_rejects_exactly(t, theError, reader2.closed),
// 		promise_rejects_exactly(t, theError, reader2.read())
// 	);
//
// 	return Promise.all(promiseAsserts);
//
// }, 'Getting a second reader after erroring the stream and releasing the reader should succeed');