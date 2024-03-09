package di

import (
	"fmt"
	"log"
	"reflect"
)

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

// Get takes a pointer or interface type argument and returns the provided implemenation.
func Get[Type any](i *Injector) Type {
	defer func() {
		if r := recover(); r != nil {
			i.handleError(fmt.Errorf("unable to resolve %s, returning empty value", reflect.TypeFor[Type]().String()))
		}
	}()

	if reflect.TypeFor[Type]().Kind() != reflect.Interface && reflect.TypeFor[Type]().Kind() != reflect.Ptr {
		i.handleError(fmt.Errorf("unable to resolve %s, type must be either a pointer or an interface, not: %s", reflect.TypeFor[Type]().String(), reflect.TypeFor[Type]().Kind()))
	}

	return i.get(reflect.TypeFor[Type](), "").(Type)
}

// NamedGet takes a pointer or interface type argument and a name string and returns the provided implemenation.
func NamedGet[Type any](i *Injector, name string) Type {
	defer func() {
		if r := recover(); r != nil {
			i.handleError(fmt.Errorf("unable to resolve %s, returning empty value", reflect.TypeFor[Type]().String()))
		}
	}()

	if reflect.TypeFor[Type]().Kind() != reflect.Interface && reflect.TypeFor[Type]().Kind() != reflect.Ptr {
		i.handleError(fmt.Errorf("unable to resolve %s, type must be either a pointer or an interface, not: %s", reflect.TypeFor[Type]().String(), reflect.TypeFor[Type]().Kind()))
	}

	return i.get(reflect.TypeFor[Type](), name).(Type)
}
