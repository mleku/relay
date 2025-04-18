// @generated Code generated by gen-atomicwrapper.

// Copyright (c) 2020-2023 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of this software
// and associated documentation files (the "Software"), to deal in the Software without
// restriction, including without limitation the rights to use, copy, modify, merge, publish,
// distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the
// Software is furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all copies or
// substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING
// BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
// NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM,
// DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package atomic

// Error is an atomic type-safe wrapper for error values.
type Error struct {
	_ nocmp // disallow non-atomic comparison

	v Value
}

var _zeroError error

// NewError creates a new Error.
func NewError(val error) *Error {
	x := &Error{}
	if val != _zeroError {
		x.Store(val)
	}
	return x
}

// Load atomically loads the wrapped error.
func (x *Error) Load() error {
	return unpackError(x.v.Load())
}

// Store atomically stores the passed error.
func (x *Error) Store(val error) {
	x.v.Store(packError(val))
}

// CompareAndSwap is an atomic compare-and-swap for error values.
func (x *Error) CompareAndSwap(old, new error) (swapped bool) {
	if x.v.CompareAndSwap(packError(old), packError(new)) {
		return true
	}

	if old == _zeroError {
		// If the old value is the empty value, then it's possible the underlying Value hasn't
		// been set and is nil, so retry with nil.
		return x.v.CompareAndSwap(nil, packError(new))
	}

	return false
}

// Swap atomically stores the given error and returns the old value.
func (x *Error) Swap(val error) (old error) {
	return unpackError(x.v.Swap(packError(val)))
}
