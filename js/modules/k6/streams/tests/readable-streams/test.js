function assert_iter_result(iterResult, value, done, message) {
	const prefix = message === undefined ? '' : `${message} `;
	assert_equals(typeof iterResult, 'object', `${prefix}type is object`);
	assert_equals(Object.getPrototypeOf(iterResult), Object.prototype, `${prefix}[[Prototype]]`);
	assert_array_equals(Object.getOwnPropertyNames(iterResult).sort(), ['done', 'value'], `${prefix}property names`);
	assert_equals(iterResult.value, value, `${prefix}value`);
	assert_equals(iterResult.done, done, `${prefix}done`);
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