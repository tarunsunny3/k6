function templatedRSEmpty(label, factory) {
	test(() => {
	}, 'Running templatedRSEmpty with ' + label);

	test(() => {

		const rs = factory();

		// FIXME: Uncomment once we add that support
		//assert_equals(typeof rs.locked, 'boolean', 'has a boolean locked getter');
		assert_equals(typeof rs.cancel, 'function', 'has a cancel method');
		assert_equals(typeof rs.getReader, 'function', 'has a getReader method');
		// FIXME: Uncomment once we add that support
		// assert_equals(typeof rs.pipeThrough, 'function', 'has a pipeThrough method');
		// assert_equals(typeof rs.pipeTo, 'function', 'has a pipeTo method');
		// assert_equals(typeof rs.tee, 'function', 'has a tee method');

	}, label + ': instances have the correct methods and properties');

	test(() => {
		const rs = factory();

		assert_throws_js(TypeError, () => rs.getReader({mode: ''}), 'empty string mode should throw');
		assert_throws_js(TypeError, () => rs.getReader({mode: null}), 'null mode should throw');
		assert_throws_js(TypeError, () => rs.getReader({mode: 'asdf'}), 'asdf mode should throw');
		assert_throws_js(TypeError, () => rs.getReader(5), '5 should throw');

		// Should not throw
		rs.getReader(null);

	}, label + ': calling getReader with invalid arguments should throw appropriate errors');
}

function templatedRSEmptyReader(label, factory) {
	test(() => {
	}, 'Running templatedRSEmptyReader with ' + label);

	test(() => {

		const reader = factory().reader;

		assert_true('closed' in reader, 'has a closed property');
		assert_equals(typeof reader.closed.then, 'function', 'closed property is thenable');

		assert_equals(typeof reader.cancel, 'function', 'has a cancel method');
		assert_equals(typeof reader.read, 'function', 'has a read method');
		assert_equals(typeof reader.releaseLock, 'function', 'has a releaseLock method');

	}, label + ': instances have the correct methods and properties');

	test(() => {

		const stream = factory().stream;

		assert_true(stream.locked, 'locked getter should return true');

	}, label + ': locked should be true');

	promise_test(t => {

		const reader = factory().reader;

		reader.read().then(
			() => assert_unreached('read() should not fulfill'),
			() => assert_unreached('read() should not reject')
		);

		return delay(500);

	}, label + ': read() should never settle');

	promise_test(t => {

		const reader = factory().reader;

		reader.read().then(
			() => assert_unreached('read() should not fulfill'),
			() => assert_unreached('read() should not reject')
		);

		reader.read().then(
			() => assert_unreached('read() should not fulfill'),
			() => assert_unreached('read() should not reject')
		);

		return delay(500);

	}, label + ': two read()s should both never settle');

	test(() => {

		const reader = factory().reader;
		assert_not_equals(reader.read(), reader.read(), 'the promises returned should be distinct');

	}, label + ': read() should return distinct promises each time');

	test(() => {

		const stream = factory().stream;
		assert_throws_js(TypeError, () => stream.getReader(), 'stream.getReader() should throw a TypeError');

	}, label + ': getReader() again on the stream should fail');

	promise_test(async t => {

		const streamAndReader = factory();
		const stream = streamAndReader.stream;
		const reader = streamAndReader.reader;

		const read1 = reader.read();
		const read2 = reader.read();
		const closed = reader.closed;

		reader.releaseLock();

		assert_false(stream.locked, 'the stream should be unlocked');

		await Promise.all([
			promise_rejects_js(t, TypeError, read1, 'first read should reject'),
			promise_rejects_js(t, TypeError, read2, 'second read should reject'),
			promise_rejects_js(t, TypeError, closed, 'closed should reject')
		]);

	}, label + ': releasing the lock should reject all pending read requests');

	promise_test(t => {

		const reader = factory().reader;
		reader.releaseLock();

		return Promise.all([
			promise_rejects_js(t, TypeError, reader.read()),
			promise_rejects_js(t, TypeError, reader.read())
		]);

	}, label + ': releasing the lock should cause further read() calls to reject with a TypeError');

	promise_test(t => {

		const reader = factory().reader;

		const closedBefore = reader.closed;
		reader.releaseLock();
		const closedAfter = reader.closed;

		assert_equals(closedBefore, closedAfter, 'the closed promise should not change identity');

		return promise_rejects_js(t, TypeError, closedBefore);

	}, label + ': releasing the lock should cause closed calls to reject with a TypeError');

	test(() => {

		const streamAndReader = factory();
		const stream = streamAndReader.stream;
		const reader = streamAndReader.reader;

		reader.releaseLock();
		assert_false(stream.locked, 'locked getter should return false');

	}, label + ': releasing the lock should cause locked to become false');

	promise_test(() => {

		const reader = factory().reader;
		reader.cancel();

		return reader.read().then(r => {
			assert_object_equals(r, {value: undefined, done: true}, 'read()ing from the reader should give a done result');
		});

	}, label + ': canceling via the reader should cause the reader to act closed');

	promise_test(t => {

		const stream = factory().stream;
		return promise_rejects_js(t, TypeError, stream.cancel());

	}, label + ': canceling via the stream should fail');
}