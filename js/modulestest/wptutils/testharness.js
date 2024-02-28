// This file contains a partial adaptation of the testharness.js implementation from
// the W3C WebCrypto API test suite. It is not intended to be a complete
// implementation, but rather a minimal set of functions to support the
// tests for this extension.
//
// Some of the function have been modified to support the k6 javascript runtime,
// and to limit its dependency to the rest of the W3C WebCrypto API test suite internal
// codebase.
//
// The original testharness.js implementation is available at:
// https://github.com/web-platform-tests/wpt/blob/3a3453c62176c97ab51cd492553c2dacd24366b1/resources/testharness.js

/**
 * Utility functions.
 */
function test(func, name)
{
	try {
		func();
	} catch(e) {
		throw `${name} failed - ${e}`;
	}
}

function promise_test(func, name)
{
	func().catch((e) => {
		throw `${name} failed - ${e}`;
	});
}

/**
 * @class
 * Exception type that represents a failing assert.
 *
 * @param {string} message - Error message.
 */
function AssertionError(message)
{
	if (typeof message == "string") {
		message = sanitize_unpaired_surrogates(message);
	}
	this.message = message;
	this.stack = get_stack();
}

AssertionError.prototype = Object.create(Error.prototype);

function assert(expected_true, function_name, description, error, substitutions)
{
	if (expected_true !== true) {
		// NOTE: This is a simplified version of the original implementation
		// found at: https://github.com/web-platform-tests/wpt/blob/e955fbc72b5a98e1c2dc6a6c1a048886c8a99785/resources/testharness.js#L4622
		// var msg = make_message(function_name, description,
		// 	error, substitutions);
		var msg = `${function_name}: ${description} ${error}`;

		throw new AssertionError(msg);
	}
}

/**
 * Assert that ``actual`` is the same value as ``expected``.
 *
 * For objects this compares by cobject identity; for primitives
 * this distinguishes between 0 and -0, and has correct handling
 * of NaN.
 *
 * @param {Any} actual - Test value.
 * @param {Any} expected - Expected value.
 * @param {string} [description] - Description of the condition being tested.
 */
function assert_equals(actual, expected, description) {
	if (actual !== expected) {
		throw `assert_equals ${description} expected (${typeof expected}) ${expected} but got (${typeof actual}) ${actual}`;
	}
}

/**
 * Assert that ``actual`` is not the same value as ``expected``.
 *
 * Comparison is as for :js:func:`assert_equals`.
 *
 * @param {Any} actual - Test value.
 * @param {Any} expected - The value ``actual`` is expected to be different to.
 * @param {string} [description] - Description of the condition being tested.
 */
function assert_not_equals(actual, expected, description) {
	if (actual === expected) {
		throw `assert_not_equals ${description} got disallowed value ${actual}`;
	}
}

/**
 * Assert that ``actual`` is strictly true
 *
 * @param {Any} actual - Value that is asserted to be true
 * @param {string} [description] - Description of the condition being tested
 */
function assert_true(actual, description) {
	if (!actual) {
		throw `assert_true ${description} expected true got ${actual}`;
	}
}

/**
 * Assert that ``actual`` is strictly false
 *
 * @param {Any} actual - Value that is asserted to be false
 * @param {string} [description] - Description of the condition being tested
 */
function assert_false(actual, description) {
	if (actual) {
		throw `assert_true ${description} expected false got ${actual}`;
	}
}

/**
 * Assert that ``expected`` is an array and ``actual`` is one of the members.
 * This is implemented using ``indexOf``, so doesn't handle NaN or ±0 correctly.
 *
 * @param {Any} actual - Test value.
 * @param {Array} expected - An array that ``actual`` is expected to
 * be a member of.
 * @param {string} [description] - Description of the condition being tested.
 */
function assert_in_array(actual, expected, description) {
	if (expected.indexOf(actual) === -1) {
		throw `assert_in_array ${description} value ${actual} not in array ${expected}`;
	}
}

/**
 * Asserts if called. Used to ensure that a specific codepath is
 * not taken e.g. that an error event isn't fired.
 *
 * @param {string} [description] - Description of the condition being tested.
 */
function assert_unreached(description) {
	throw `reached unreachable code, reason: ${description}`
}

/**
 * Assert that ``actual`` and ``expected`` are both arrays, and that the array properties of
 * ``actual`` and ``expected`` are all the same value (as for :js:func:`assert_equals`).
 *
 * @param {Array} actual - Test array.
 * @param {Array} expected - Array that is expected to contain the same values as ``actual``.
 * @param {string} [description] - Description of the condition being tested.
 */
function assert_array_equals(actual, expected, description) {
	if (typeof actual !== "object" || actual === null || !("length" in actual)) {
		throw `assert_array_equals ${description} value is ${actual}, expected array`;
	}

	if (actual.length !== expected.length) {
		throw `assert_array_equals ${description} lengths differ, expected array ${expected} length ${expected.length}, got ${actual} length ${actual.length}`;
	}

	for (var i = 0; i < actual.length; i++) {
		if (actual.hasOwnProperty(i) !== expected.hasOwnProperty(i)) {
			throw `assert_array_equals ${description} expected property ${i} to be ${expected.hasOwnProperty(i)} but was ${actual.hasOwnProperty(i)} (expected array ${expected} got ${actual})`;
		}

		if (!same_value(expected[i], actual[i])) {
			throw `assert_array_equals ${description} expected property ${i} to be ${expected[i]} but got ${actual[i]} (expected array ${expected} got ${actual})`;
		}
	}
}

/**
 * Assert the provided value is thrown.
 *
 * @param {value} exception The expected exception.
 * @param {Function} func Function which should throw.
 * @param {string} [description] Error description for the case that the error is not thrown.
 */
function assert_throws_exactly(exception, func, description)
{
	assert_throws_exactly_impl(exception, func, description,
		"assert_throws_exactly");
}

/**
 * Like assert_throws_exactly but allows specifying the assertion type
 * (assert_throws_exactly or promise_rejects_exactly, in practice).
 */
function assert_throws_exactly_impl(exception, func, description,
									assertion_type)
{
	try {
		func.call(this);
		assert(false, assertion_type, description,
			"${func} did not throw", {func:func});
	} catch (e) {
		if (e instanceof AssertionError) {
			throw e;
		}

		assert(same_value(e, exception), assertion_type, description,
			"${func} threw ${e} but we expected it to throw ${exception}",
			{func:func, e:e, exception:exception});
	}
}

/**
 * Assert that a Promise is rejected with the provided value.
 *
 * @param {Test} test - the `Test` to use for the assertion.
 * @param {Any} exception - The expected value of the rejected promise.
 * @param {Promise} promise - The promise that's expected to
 * reject.
 * @param {string} [description] Error message to add to assert in case of
 *                               failure.
 */
function promise_rejects_exactly(test, exception, promise, description) {
	return promise.then(
		() => {
			throw new Error("Should have rejected: " + description);
		},
		(e) => {
			assert_throws_exactly_impl(exception, function() { throw e},
				description, "promise_rejects_exactly");
		}
	);
}

/**
 * Assert a JS Error with the expected constructor is thrown.
 *
 * @param {object} constructor The expected exception constructor.
 * @param {Function} func Function which should throw.
 * @param {string} [description] Error description for the case that the error is not thrown.
 */
function assert_throws_js(constructor, func, description)
{
	assert_throws_js_impl(constructor, func, description,
		"assert_throws_js");
}

/**
 * Like assert_throws_js but allows specifying the assertion type
 * (assert_throws_js or promise_rejects_js, in practice).
 */
function assert_throws_js_impl(constructor, func, description,
							   assertion_type)
{
	try {
		func.call(this);
		assert(false, assertion_type, description,
			"${func} did not throw", {func:func});
	} catch (e) {
		if (e instanceof AssertionError) {
			throw e;
		}

		// Basic sanity-checks on the thrown exception.
		assert(typeof e === "object",
			assertion_type, description,
			"${func} threw ${e} with type ${type}, not an object",
			{func:func, e:e, type:typeof e});

		assert(e !== null,
			assertion_type, description,
			"${func} threw null, not an object",
			{func:func});

		// Note @oleiade: As k6 does not throw error objects that match the Javascript
		// standard errors and their associated expectations and properties, we cannot
		// rely on the WPT assertions to be true.
		//
		// Instead, we check that the error object has the shape we give it when we throw it.
		// Namely that it has a name property that matches the name of the expected constructor.
		assert('value' in e,
			assertion_type, description,
			"${func} threw ${e} without a value property",
			{func:func, e:e});

		assert('name' in e.value,
			assertion_type, description,
			"${func} threw ${e} without a name property",
			{func:func, e:e});

		assert(e.value.name === constructor.name,
			assertion_type, description,
			"${func} threw ${e} with name ${e.name}, not ${constructor.name}",
			{func:func, e:e, constructor:constructor});

		// Note @oleiade: We deactivated the following assertions in favor of our own
		// as mentioned above.

		// Basic sanity-check on the passed-in constructor
		// assert(typeof constructor == "function",
		// 	assertion_type, description,
		// 	"${constructor} is not a constructor",
		// 	{constructor:constructor});
		// var obj = constructor;
		// while (obj) {
		// 	if (typeof obj === "function" &&
		// 		obj.name === "Error") {
		// 		break;
		// 	}
		// 	obj = Object.getPrototypeOf(obj);
		// }
		// assert(obj != null,
		// 	assertion_type, description,
		// 	"${constructor} is not an Error subtype",
		// 	{constructor:constructor});
		//
		// // And checking that our exception is reasonable
		// assert(e.constructor === constructor &&
		// 	e.name === constructor.name,
		// 	assertion_type, description,
		// 	"${func} threw ${actual} (${actual_name}) expected instance of ${expected} (${expected_name})",
		// 	{func:func, actual:e, actual_name:e.name,
		// 		expected:constructor,
		// 		expected_name:constructor.name});
	}
}

function same_value(x, y) {
	if (y !== y) {
		//NaN case
		return x !== x;
	}
	if (x === 0 && y === 0) {
		//Distinguish +0 and -0
		return 1 / x === 1 / y;
	}
	return x === y;
}

function code_unit_str(char) {
	return 'U+' + char.charCodeAt(0).toString(16);
}

function sanitize_unpaired_surrogates(str) {
	return str.replace(
		/([\ud800-\udbff]+)(?![\udc00-\udfff])|(^|[^\ud800-\udbff])([\udc00-\udfff]+)/g,
		function(_, low, prefix, high) {
			var output = prefix || "";  // prefix may be undefined
			var string = low || high;  // only one of these alternates can match
			for (var i = 0; i < string.length; i++) {
				output += code_unit_str(string[i]);
			}
			return output;
		});
}

const get_stack = function() {
	var stack = new Error().stack;

	// 'Error.stack' is not supported in all browsers/versions
	if (!stack) {
		return "(Stack trace unavailable)";
	}

	var lines = stack.split("\n");

	// Create a pattern to match stack frames originating within testharness.js.  These include the
	// script URL, followed by the line/col (e.g., '/resources/testharness.js:120:21').
	// Escape the URL per http://stackoverflow.com/questions/3561493/is-there-a-regexp-escape-function-in-javascript
	// in case it contains RegExp characters.
	// NOTE @oleiade: We explicitly bypass the get_script_url operation as it's specific to the
	// web platform test suite and enforce the use of an empty string instead.
	// var script_url = get_script_url();
	var script_url = '';
	var re_text = script_url ? script_url.replace(/[-\/\\^$*+?.()|[\]{}]/g, '\\$&') : "\\btestharness.js";
	var re = new RegExp(re_text + ":\\d+:\\d+");

	// Some browsers include a preamble that specifies the type of the error object.  Skip this by
	// advancing until we find the first stack frame originating from testharness.js.
	var i = 0;
	while (!re.test(lines[i]) && i < lines.length) {
		i++;
	}

	// Then skip the top frames originating from testharness.js to begin the stack at the test code.
	while (re.test(lines[i]) && i < lines.length) {
		i++;
	}

	// Paranoid check that we didn't skip all frames.  If so, return the original stack unmodified.
	if (i >= lines.length) {
		return stack;
	}

	return lines.slice(i).join("\n");
}
