// Original source file: https://github.com/web-platform-tests/wpt/blob/3fd901dba4d461afda4cf9b692f8bd99fb05f4e1/streams/readable-streams/templated.any.js
// META: global=window,worker,shadowrealm
// META: script=../resources/test-utils.js
// META: script=../resources/rs-test-templates.js
'use strict';

// Run the readable stream test templates against readable streams created directly using the constructor

const theError = { name: 'boo!' };
const chunks = ['a', 'b'];

templatedRSEmpty('ReadableStream (empty)', () => {
	return new ReadableStream();
});

templatedRSEmptyReader('ReadableStream (empty) reader', () => {
	return streamAndDefaultReader(new ReadableStream());
});

function streamAndDefaultReader(stream) {
	return { stream, reader: stream.getReader() };
}