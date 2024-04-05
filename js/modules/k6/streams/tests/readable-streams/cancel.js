// Original source file: https://github.com/web-platform-tests/wpt/blob/3fd901dba4d461afda4cf9b692f8bd99fb05f4e1/streams/readable-streams/cancel.any.js
// META: global=window,worker,shadowrealm
// META: script=../resources/test-utils.js
// META: script=../resources/rs-utils.js
'use strict';

promise_test(t => {

	const randomSource = new RandomPushSource();

	let cancellationFinished = false;
	const rs = new ReadableStream({
		start(c) {
			randomSource.ondata = c.enqueue.bind(c);
			randomSource.onend = c.close.bind(c);
			randomSource.onerror = c.error.bind(c);
		},

		pull() {
			randomSource.readStart();
		},

		cancel() {
			randomSource.readStop();

			return new Promise(resolve => {
				step_timeout(() => {
					cancellationFinished = true;
					resolve();
				}, 1);
			});
		}
	});

	const reader = rs.getReader();

	// We call delay multiple times to avoid cancelling too early for the
	// source to enqueue at least one chunk.
	const cancel = delay(5).then(() => delay(5)).then(() => delay(5)).then(() => {
		const cancelPromise = reader.cancel();
		assert_false(cancellationFinished, 'cancellation in source should happen later');
		return cancelPromise;
	});

	return readableStreamToArray(rs, reader).then(chunks => {
		assert_greater_than(chunks.length, 0, 'at least one chunk should be read');
		for (let i = 0; i < chunks.length; i++) {
			assert_equals(chunks[i].length, 128, 'chunk ' + i + ' should have 128 bytes');
		}
		return cancel;
	}).then(() => {
		assert_true(cancellationFinished, 'it returns a promise that is fulfilled when the cancellation finishes');
	});

}, 'ReadableStream cancellation: integration test on an infinite stream derived from a random push source');