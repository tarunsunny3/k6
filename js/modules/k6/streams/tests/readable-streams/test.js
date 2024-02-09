function assert_iter_result(iterResult, value, done, message) {
	const prefix = message === undefined ? '' : `${message} `;
	assert_equals(typeof iterResult, 'object', `${prefix}type is object`);
	assert_equals(Object.getPrototypeOf(iterResult), Object.prototype, `${prefix}[[Prototype]]`);
	assert_array_equals(Object.getOwnPropertyNames(iterResult).sort(), ['done', 'value'], `${prefix}property names`);
	assert_equals(iterResult.value, value, `${prefix}value`);
	assert_equals(iterResult.done, done, `${prefix}done`);
}

// NOTE: Async generators are not supported yet
// test(() => {
// 	const s = new ReadableStream();
// 	const it = s.values();
// 	const proto = Object.getPrototypeOf(it);
//
// 	const AsyncIteratorPrototype = Object.getPrototypeOf(Object.getPrototypeOf(async function* () {}).prototype);
// 	assert_equals(Object.getPrototypeOf(proto), AsyncIteratorPrototype, 'prototype should extend AsyncIteratorPrototype');
//
// 	const methods = ['next', 'return'].sort();
// 	assert_array_equals(Object.getOwnPropertyNames(proto).sort(), methods, 'should have all the correct methods');
//
// 	for (const m of methods) {
// 		const propDesc = Object.getOwnPropertyDescriptor(proto, m);
// 		assert_true(propDesc.enumerable, 'method should be enumerable');
// 		assert_true(propDesc.configurable, 'method should be configurable');
// 		assert_true(propDesc.writable, 'method should be writable');
// 		assert_equals(typeof it[m], 'function', 'method should be a function');
// 		assert_equals(it[m].name, m, 'method should have the correct name');
// 	}
//
// 	assert_equals(it.next.length, 0, 'next should have no parameters');
// 	assert_equals(it.return.length, 1, 'return should have 1 parameter');
// 	assert_equals(typeof it.throw, 'undefined', 'throw should not exist');
// }, 'Async iterator instances should have the correct list of properties');

// NOTE: for await (const chunk of s) {...} is not supported yet
promise_test(async () => {
	const s = new ReadableStream({
		start(c) {
			c.enqueue(1);
			c.enqueue(2);
			c.enqueue(3);
			c.close();
		}
	});

	const chunks = [];
	const reader = s.getReader();
	while (true) {
		const {value: chunk, done} = await reader.read();
		if (done) {
			break;
		}
		chunks.push(chunk);
	}
	assert_array_equals(chunks, [1, 2, 3]);
}, 'Async-iterating a push source');

// NOTE: GoError: AssertionError:stream does not have a default reader
// promise_test(async () => {
// 	let i = 1;
// 	const s = new ReadableStream({
// 		pull(c) {
// 			c.enqueue(i);
// 			if (i >= 3) {
// 				c.close();
// 			}
// 			i += 1;
// 		}
// 	});
//
// 	const chunks = [];
// 	const reader = s.getReader();
// 	while (true) {
// 		const {value: chunk, done} = await reader.read();
// 		if (done) {
// 			break;
// 		}
// 		chunks.push(chunk);
// 	}
// 	assert_array_equals(chunks, [1, 2, 3]);
// }, 'Async-iterating a pull source');

promise_test(async () => {
	const s = new ReadableStream({
		start(c) {
			c.enqueue(undefined);
			c.enqueue(undefined);
			c.enqueue(undefined);
			c.close();
		}
	});

	const chunks = [];
	const reader = s.getReader();
	while (true) {
		const {value: chunk, done} = await reader.read();
		if (done) {
			break;
		}
		chunks.push(chunk);
	}
	assert_array_equals(chunks, [undefined, undefined, undefined]);
}, 'Async-iterating a push source with undefined values');

// NOTE: GoError: AssertionError:stream does not have a default reader
// promise_test(async () => {
// 	let i = 1;
// 	const s = new ReadableStream({
// 		pull(c) {
// 			c.enqueue(undefined);
// 			if (i >= 3) {
// 				c.close();
// 			}
// 			i += 1;
// 		}
// 	});
//
// 	const chunks = [];
// 	const reader = s.getReader();
// 	while (true) {
// 		const {value: chunk, done} = await reader.read();
// 		if (done) {
// 			break;
// 		}
// 		chunks.push(chunk);
// 	}
// 	assert_array_equals(chunks, [undefined, undefined, undefined]);
// }, 'Async-iterating a pull source with undefined values');

// TODO: Move this to a separate file
const recordingReadableStream = (extras = {}, strategy) => {
	let controllerToCopyOver;
	const stream = new ReadableStream({
		type: extras.type,
		start(controller) {
			controllerToCopyOver = controller;

			if (extras.start) {
				return extras.start(controller);
			}

			return undefined;
		},
		pull(controller) {
			stream.events.push('pull');

			if (extras.pull) {
				return extras.pull(controller);
			}

			return undefined;
		},
		cancel(reason) {
			stream.events.push('cancel', reason);
			stream.eventsWithoutPulls.push('cancel', reason);

			if (extras.cancel) {
				return extras.cancel(reason);
			}

			return undefined;
		}
	}, strategy);

	stream.controller = controllerToCopyOver;
	stream.events = [];
	stream.eventsWithoutPulls = [];

	return stream;
};

// NOTE: ReferenceError: CountQueuingStrategy is not defined
// promise_test(async () => {
// 	let i = 1;
// 	const s = recordingReadableStream({
// 		pull(c) {
// 			c.enqueue(i);
// 			if (i >= 3) {
// 				c.close();
// 			}
// 			i += 1;
// 		},
// 	}, new CountQueuingStrategy({ highWaterMark: 0 }));
//
// 	const it = s.values();
// 	assert_array_equals(s.events, []);
//
// 	const read1 = await it.next();
// 	assert_iter_result(read1, 1, false);
// 	assert_array_equals(s.events, ['pull']);
//
// 	const read2 = await it.next();
// 	assert_iter_result(read2, 2, false);
// 	assert_array_equals(s.events, ['pull', 'pull']);
//
// 	const read3 = await it.next();
// 	assert_iter_result(read3, 3, false);
// 	assert_array_equals(s.events, ['pull', 'pull', 'pull']);
//
// 	const read4 = await it.next();
// 	assert_iter_result(read4, undefined, true);
// 	assert_array_equals(s.events, ['pull', 'pull', 'pull']);
// }, 'Async-iterating a pull source manually');

// NOTE: TypeError: could not convert function call parameter 0: could not convert e to error
// promise_test(async () => {
// 	const s = new ReadableStream({
// 		start(c) {
// 			c.error('e');
// 		},
// 	});
//
// 	try {
// 		const reader = s.getReader();
// 		while (true) {
// 			const {done} = await reader.read();
// 			if (done) {
// 				break;
// 			}
// 		}
// 		assert_unreached();
// 	} catch (e) {
// 		assert_equals(e, 'e');
// 	}
// }, 'Async-iterating an errored stream throws');

promise_test(async () => {
	const s = new ReadableStream({
		start(c) {
			c.close();
		}
	});

	// TODO: Abstract as a function?
	const reader = s.getReader();
	while (true) {
		const {done} = await reader.read();
		if (done) {
			break;
		}
		assert_unreached();
	}
}, 'Async-iterating a closed stream never executes the loop body, but works fine');

// NOTE: flushAsyncEvents requires extra test harness support (setTimeout)
// promise_test(async () => {
// 	const s = new ReadableStream();
//
// 	const loop = async () => {
// 		const reader = s.getReader();
// 		while (true) {
// 			const {done} = await reader.read();
// 			if (done) {
// 				break;
// 			}
// 			assert_unreached();
// 		}
// 		assert_unreached();
// 	};
//
// 	await Promise.race([
// 		loop(),
// 		flushAsyncEvents()
// 	]);
// }, 'Async-iterating an empty but not closed/errored stream never executes the loop body and stalls the async function');

promise_test(async () => {
	const s = new ReadableStream({
		start(c) {
			c.enqueue(1);
			c.enqueue(2);
			c.enqueue(3);
			c.close();
		},
	});

	const reader = s.getReader();
	const readResult = await reader.read();
	assert_iter_result(readResult, 1, false);
	reader.releaseLock();

	const chunks = [];
	while (true) {
		const {value: chunk, done} = await reader.read();
		if (done) {
			break;
		}
		chunks.push(chunk);
	}
	assert_array_equals(chunks, [2, 3]);
}, 'Async-iterating a partially consumed stream');


for (const type of ['throw', 'break', 'return']) {
	for (const preventCancel of [false, true]) {
		promise_test(async () => {
			const s = recordingReadableStream({
				start(c) {
					c.enqueue(0);
				}
			});

			// use a separate function for the loop body so return does not stop the test
			const loop = async () => {
				const iterator = s.values({preventCancel});
				while (true) {
					const {done} = await iterator.next();
					if (done) {
						break;
					}

					if (type === 'throw') {
						throw new Error();
					} else if (type === 'break') {
						break;
					} else if (type === 'return') {
						return;
					}
				}
			};

			try {
				await loop();
			} catch (e) {
			}

			if (preventCancel) {
				assert_array_equals(s.events, ['pull'], `cancel() should not be called`);
			} else {
				assert_array_equals(s.events, ['pull', 'cancel', undefined], `cancel() should be called`);
			}
		}, `Cancellation behavior when ${type}ing inside loop body; preventCancel = ${preventCancel}`);
	}
}

async function run_test() {
	assert_not_equals(new ReadableStream(), undefined, "constructor");

	const func = async () => {
		const s = new ReadableStream({
			start(c) {
				c.enqueue(1);
				c.enqueue(2);
				c.enqueue(3);
				c.close();
			},
		});

		// read the first two chunks, then release lock
		const chunks = [];
		const reader = s.getReader();
		while (true) {
			const {value: chunk, done} = await reader.read();
			chunks.push(chunk);
			if (done || chunk >= 2) {
				break;
			}
		}
		assert_array_equals(chunks, [1, 2]);

		const readResult = await reader.read();
		assert_iter_result(readResult, 3, false);
		await reader.closed;
	};

	await func();
}