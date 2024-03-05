// MIT License

// Copyright (c) 2019 GoLobby

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

// original source code: github.com/golobby/container

package di

import "log"

// GlobalInjector is the global repository of bindings
var GlobalInjector = NewInjector()

// SetErrorHandler sets the global error handler
func SetErrorHandler(handler errorHandler) {
	GlobalInjector.SetErrorHandler(handler)
}

// SetLogger attaches a custom logger.
func SetLogger(logger log.Logger) {
	GlobalInjector.SetLogger(logger)
}

// EnableDebugLogging enables debug logging.
func EnableDebugLogging() {
	GlobalInjector.EnableDebugLogging()
}

// DisableDebugLogging disables debug logging.
func DisableDebugLogging() {
	GlobalInjector.DisableDebugLogging()
}

// Singleton binds an abstraction to concrete for further singleton resolves.
// It takes a resolver function that returns the concrete, and its return type matches the abstraction (interface).
// The resolver function can have arguments of abstraction that have been declared in the Injector already.
func Singleton(resolver interface{}) {
	GlobalInjector.Singleton(resolver)
}

// NamedSingleton binds like the Singleton method but for named bindings.
func NamedSingleton(name string, resolver interface{}) {
	GlobalInjector.NamedSingleton(name, resolver)
}

// Instance binds an abstraction to concrete for further transient resolves.
// It takes a resolver function that returns the concrete, and its return type matches the abstraction (interface).
// The resolver function can have arguments of abstraction that have been declared in the Injector already.
func Instance(resolver interface{}) {
	GlobalInjector.Instance(resolver)
}

// NamedInstance binds like the Instance method but for named bindings.
func NamedInstance(name string, resolver interface{}) {
	GlobalInjector.NamedInstance(name, resolver)
}

// Reset deletes all the existing bindings and empties the container instance.
func Reset() {
	GlobalInjector.Reset()
}

// Call takes a function (receiver) with one or more arguments of the abstractions (interfaces).
// It invokes the function (receiver) and passes the related implementations.
func Call(receiver interface{}) {
	GlobalInjector.Call(receiver)
}

// Resolve takes an abstraction (interface reference) and fills it with the related implementation.
func Resolve(abstraction interface{}) {
	GlobalInjector.Resolve(abstraction)
}

// NamedResolve resolves like the Resolve method but for named bindings.
func NamedResolve(abstraction interface{}, name string) {
	GlobalInjector.NamedResolve(abstraction, name)
}

// Fill takes a struct and resolves the fields with the tag `container:"inject"`
func Fill(receiver interface{}) {
	GlobalInjector.Fill(receiver)
}
