package di

import (
	"errors"
	"fmt"
	"log"
	"reflect"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/fatih/color"
)

const (
	tagName         = "di"
	injectByType    = "type"
	injectByName    = "name"
	bindingPrefix   = "BINDING"
	resolvingPrefix = "RESOLVING"
	returningPrefix = "RETURNING"
	invokingPrefix  = "INVOKING"
	fillingPrefix   = "FILLING"
)

type bindingtype int

const (
	Binding_Instance bindingtype = iota
	Binding_Singleton
)

// binding holds a binding provider and an instance (for singleton bindings).
type binding struct {
	provider interface{} // provider function that creates the appropriate implementation of the related abstraction
	mu       *sync.Mutex // mutex for retrieving a singleton at evaluation time
	instance interface{} // instance stored for reusing in singleton bindings
	btype    bindingtype // type of the binding (singleton or instance)
}

// resolve creates an appropriate implementation of the related abstraction
func (b *binding) resolve(injector *Injector, name string, instantiated map[reflect.Type]map[string]interface{}) (interface{}, error) {

	providerType := reflect.TypeOf(b.provider)

	if injector.isVerbose() {
		injector.incrementLoggerIndent()
		defer injector.decrementLoggerIndent()

		injector.logDebug(fmt.Sprintf("%s: provider for type `%s`", color.MagentaString(resolvingPrefix), color.BlueString(fullyQualifiedTypeString(providerType.Out(0)))))
	}

	// resolve circular dependencies within a resolution call
	instances, instantiatedTypeAlready := instantiated[providerType]
	if !instantiatedTypeAlready {
		instantiated[providerType] = make(map[string]interface{})
		instances, _ = instantiated[providerType]
	}

	instance, instantiatedAlready := instances[name]
	if instantiatedAlready {
		return instance, nil
	}

	if b.btype == Binding_Singleton {
		// we may have two callers try to resolve the singleton at once, which could create two instances of it
		// access the lock before checking if an instance is defined. If it is, release lock and return
		// otherwise, create a new one, set it, release the lock and return
		if injector.isVerbose() {
			injector.logDebug(fmt.Sprintf("%s: attempting to access instance for singleton `%s`", color.MagentaString(returningPrefix), color.YellowString(fmt.Sprintf("%+v", b))))
		}

		b.mu.Lock()
		defer b.mu.Unlock()
		if b.instance == nil {

			if injector.isVerbose() {
				injector.logDebug(fmt.Sprintf("%s: invoking provider to create singleton instance", color.MagentaString(returningPrefix)))
			}

			instance, err := injector.invoke(b.provider)
			if err != nil {
				return nil, err
			}

			instances[name] = instance

			err = injector.fill(instance, instantiated)
			if err != nil {
				return nil, err
			}

			b.instance = instance
		}

		return b.instance, nil
	}

	instance, err := injector.invoke(b.provider)
	if err != nil {
		return nil, err
	}

	instances[name] = instance

	err = injector.fill(instance, instantiated)
	if err != nil {
		return nil, err
	}
	return instance, nil
}

type errorHandler func(error)

// Injector holds all of the declared bindings
type Injector struct {
	bindings      map[reflect.Type]map[string]*binding
	verbose       int32
	verboseIndent int32
	errHandler    errorHandler
	logger        *log.Logger
	mu            *sync.RWMutex
}

// NewInjector creates a new instance of the Injector
func NewInjector() *Injector {
	return &Injector{
		bindings:      make(map[reflect.Type]map[string]*binding),
		mu:            &sync.RWMutex{},
		verbose:       0,
		verboseIndent: 0,
		errHandler:    nil,
	}
}

// SetLogger sets the injectors logger
func (injector *Injector) SetLogger(logger log.Logger) {
	injector.mu.Lock()
	defer injector.mu.Unlock()
	injector.logger = &logger
}

// SetErrorHandler sets the injectors error handler
func (injector *Injector) SetErrorHandler(handler errorHandler) {
	injector.mu.Lock()
	defer injector.mu.Unlock()

	injector.errHandler = handler
}

func (injector *Injector) handleError(err error) {
	injector.mu.RLock()
	defer injector.mu.RUnlock()

	if injector.errHandler != nil {
		injector.errHandler(err)
	} else {
		panic(err)
	}
}

func (injector *Injector) incrementLoggerIndent() {
	atomic.AddInt32(&injector.verboseIndent, 1)
}

func (injector *Injector) decrementLoggerIndent() {
	atomic.AddInt32(&injector.verboseIndent, -1)
}

// EnableDebugLogging enables debug logging.
func (injector *Injector) EnableDebugLogging() {
	atomic.StoreInt32(&injector.verbose, int32(1))
}

// DisableDebugLogging disables debug logging.
func (injector *Injector) DisableDebugLogging() {
	atomic.StoreInt32(&injector.verbose, int32(0))
}

func (injector *Injector) isVerbose() bool {
	return atomic.LoadInt32(&injector.verbose) != 0
}

func debugTypeString(arg interface{}) string {
	typeOf := reflect.TypeOf(arg)
	if typeOf == nil {
		return "nil interface"
	}
	return fullyQualifiedTypeString(typeOf)
}

func debugNameString(arg interface{}) string {
	typeOf := reflect.TypeOf(arg)
	if typeOf == nil {
		return color.RedString("nil") + color.CyanString(") (") + color.YellowString("You need to pass by reference: i.e. injector.Resolve(&arg)")
	}
	return fullyQualifiedTypeString(typeOf)
}

func (injector *Injector) getLogPrefix() string {
	prefix := "di: "
	indent := atomic.LoadInt32(&injector.verboseIndent)
	for i := int32(0); i < indent; i++ {
		if i == indent-1 {
			prefix += "╰-> "
		} else {
			prefix += "    "
		}
	}
	return prefix
}

func (injector *Injector) errorMiddleWare(err error) error {
	if injector.isVerbose() {
		injector.incrementLoggerIndent()
		defer injector.decrementLoggerIndent()

		errStr := fmt.Sprintf("%s: %s", color.RedString("ERROR"), err.Error())

		injector.mu.RLock()
		defer injector.mu.RUnlock()

		if injector.logger != nil {
			injector.logger.Print(injector.getLogPrefix() + errStr)
		} else {
			fmt.Println(injector.getLogPrefix() + errStr)
		}
	}
	return err
}

func (injector *Injector) logDebug(str string) {
	injector.mu.RLock()
	defer injector.mu.RUnlock()

	if injector.logger != nil {
		injector.logger.Print(injector.getLogPrefix() + str)
	} else {
		fmt.Println(injector.getLogPrefix() + str)
	}
}

func (injector *Injector) get(typ reflect.Type, name string) interface{} {

	concrete, exist := injector.bindings[typ][name]
	if !exist {
		injector.handleError(fmt.Errorf("no provider found for argument of type `%s`, ensure the type provided matches the return value of the provider", fullyQualifiedTypeString(typ)))
		return nil
	}

	instance, err := concrete.resolve(injector, name, map[reflect.Type]map[string]interface{}{})
	if err != nil {
		injector.handleError(err)
		return nil
	}

	return instance
}

// bind maps an abstraction to a concrete and sets an instance if it's a singleton binding.
func (injector *Injector) bind(provider interface{}, name string, singleton bool) error {
	if injector.isVerbose() {
		injector.incrementLoggerIndent()
		defer injector.decrementLoggerIndent()
	}

	providerType := reflect.TypeOf(provider)
	if providerType.Kind() != reflect.Func {
		return injector.errorMiddleWare(errors.New("provider argument must be a function"))
	}

	if providerType.NumIn() != 0 {
		return injector.errorMiddleWare(fmt.Errorf("provider function signature of `%s` is invalid, arguments are not permitted to providers", fullyQualifiedTypeString(providerType)))
	}

	if providerType.NumOut() != 1 && providerType.NumOut() != 2 {
		return injector.errorMiddleWare(fmt.Errorf("provider function signature of `%s` is invalid, must return one or two values", fullyQualifiedTypeString(providerType)))
	}

	if providerType.Out(0).Kind() != reflect.Ptr && providerType.Out(0).Kind() != reflect.Interface {
		return injector.errorMiddleWare(fmt.Errorf("provider function signature of `%s` is invalid, must return a pointer or interface type", fullyQualifiedTypeString(providerType)))
	}

	for i := 0; i < providerType.NumOut(); i++ {
		if _, exist := injector.bindings[providerType.Out(i)]; !exist {
			injector.bindings[providerType.Out(i)] = make(map[string]*binding)
		}

		if injector.isVerbose() && i == 0 {
			injector.logDebug(fmt.Sprintf("%s: provider for type `%s` with structure `%s`", color.MagentaString(bindingPrefix), color.BlueString(fullyQualifiedTypeString(providerType.Out(i))), color.GreenString(fullyQualifiedTypeString(providerType))))
		}

		if singleton {
			if injector.isVerbose() && i == 0 {
				injector.logDebug(fmt.Sprintf("%s: singleton provider for type `%s` with structure `%s`", color.MagentaString(bindingPrefix), color.BlueString(fullyQualifiedTypeString(providerType.Out(i))), color.GreenString(fullyQualifiedTypeString(providerType))))
			}

			injector.bindings[providerType.Out(i)][name] = &binding{provider: provider, mu: &sync.Mutex{}, btype: Binding_Singleton}
		} else {
			if injector.isVerbose() && i == 0 {
				injector.logDebug(fmt.Sprintf("%s: instance provider for type `%s` with structure `%s`", color.MagentaString(bindingPrefix), color.BlueString(fullyQualifiedTypeString(providerType.Out(i))), color.GreenString(fullyQualifiedTypeString(providerType))))
			}
			injector.bindings[providerType.Out(i)][name] = &binding{provider: provider, btype: Binding_Instance}
		}
	}

	return nil
}

// invoke calls a function and returns the yielded value.
// It only works for functions that return a single value.
func (injector *Injector) invoke(function interface{}) (interface{}, error) {
	functionType := reflect.TypeOf(function)
	if injector.isVerbose() {
		injector.logDebug(fmt.Sprintf("%s: arguments for provider `%s`", color.MagentaString(resolvingPrefix), color.GreenString(fullyQualifiedTypeString(functionType))))
	}

	if injector.isVerbose() {
		injector.logDebug(fmt.Sprintf("%s: provider `%s` for type `%s`", color.MagentaString(invokingPrefix), color.GreenString(fullyQualifiedTypeString(functionType)), color.BlueString(fullyQualifiedTypeString(functionType.Out(0)))))
	}

	// no args
	var args []reflect.Value

	if functionType.NumOut() == 1 {
		if injector.isVerbose() {
			injector.incrementLoggerIndent()
		}

		res := reflect.ValueOf(function).Call(args)[0].Interface()

		if injector.isVerbose() {
			injector.decrementLoggerIndent()
		}

		resv := reflect.ValueOf(res)
		if resv.Kind() != reflect.Struct && (res == nil || resv.IsNil()) {
			return nil, injector.errorMiddleWare(fmt.Errorf("provider function returned a nil value"))
		}

		if injector.isVerbose() {
			injector.logDebug(fmt.Sprintf("%s: value %s", color.MagentaString(returningPrefix), color.YellowString(fmt.Sprintf("%+v", res))))
		}

		return res, nil
	} else if functionType.NumOut() == 2 {
		if injector.isVerbose() {
			injector.incrementLoggerIndent()
		}

		values := reflect.ValueOf(function).Call(args)
		res := values[0].Interface()
		var e error
		if values[1].Interface() != nil {
			e = values[1].Interface().(error)
		}

		if injector.isVerbose() {
			injector.decrementLoggerIndent()
		}
		if e != nil {
			if injector.isVerbose() {
				injector.logDebug(fmt.Sprintf("%s: value %s", color.MagentaString(returningPrefix), color.RedString(fmt.Sprintf("%+v", e))))
			}
			return nil, injector.errorMiddleWare(e)
		}

		resv := reflect.ValueOf(res)
		if resv.Kind() != reflect.Struct && (res == nil || resv.IsNil()) {
			return nil, injector.errorMiddleWare(fmt.Errorf("provider function returned a nil value"))
		}

		if injector.isVerbose() {
			injector.logDebug(fmt.Sprintf("%s: value %s", color.MagentaString(returningPrefix), color.YellowString(fmt.Sprintf("%+v", res))))
		}

		return res, nil
	}

	return nil, injector.errorMiddleWare(errors.New("provider function signature is invalid, provider must return one or two values"))
}

// arguments returns resolved arguments of a function.
func (injector *Injector) arguments(function interface{}) ([]reflect.Value, error) {
	functionType := reflect.TypeOf(function)
	argumentsCount := functionType.NumIn()
	arguments := make([]reflect.Value, argumentsCount)

	for i := 0; i < argumentsCount; i++ {
		abstraction := functionType.In(i)

		concrete, exist := injector.bindings[abstraction][""]
		if !exist {
			return nil, injector.errorMiddleWare(fmt.Errorf("no provider found for type `%s`", fullyQualifiedTypeString(abstraction)))
		}

		instance, err := concrete.resolve(injector, "", map[reflect.Type]map[string]interface{}{})
		if err != nil {
			return nil, err
		}
		arguments[i] = reflect.ValueOf(instance)
	}

	return arguments, nil
}

// Singleton binds an abstraction to concrete for further singleton resolves.
// It takes a provider function that returns the concrete, and its return type matches the abstraction (interface).
// The provider function can have arguments of abstraction that have been declared in the Injector already.
func (injector *Injector) Singleton(provider interface{}) *Injector {
	if injector.isVerbose() {
		injector.logDebug(fmt.Sprintf("%s%s%s", color.CyanString("Singleton("), color.GreenString(debugTypeString(provider)), color.CyanString(")")))
	}

	err := injector.bind(provider, "", true)
	if err != nil {
		injector.handleError(err)
	}

	return injector
}

// NamedSingleton binds like the Singleton method but for named bindings.
func (injector *Injector) NamedSingleton(name string, provider interface{}) *Injector {
	if injector.isVerbose() {
		injector.logDebug(fmt.Sprintf("%s%s, %s%s", color.CyanString("NamedSingleton("), color.YellowString(fmt.Sprintf("\"%s\"", name)), color.GreenString(debugTypeString(provider)), color.CyanString(")")))
	}

	err := injector.bind(provider, name, true)
	if err != nil {
		injector.handleError(err)
	}

	return injector
}

// Instance binds an abstraction to concrete for further transient resolves.
// It takes a provider function that returns the concrete, and its return type matches the abstraction (interface).
// The provider function can have arguments of abstraction that have been declared in the Injector already.
func (injector *Injector) Instance(provider interface{}) *Injector {
	if injector.isVerbose() {
		injector.logDebug(fmt.Sprintf("%s%s%s", color.CyanString("Instance("), color.GreenString(debugTypeString(provider)), color.CyanString(")")))
	}

	err := injector.bind(provider, "", false)
	if err != nil {
		injector.handleError(err)
	}

	return injector
}

// NamedInstance binds like the Instance method but for named bindings.
func (injector *Injector) NamedInstance(name string, provider interface{}) *Injector {
	if injector.isVerbose() {
		injector.logDebug(fmt.Sprintf("%s%s, %s%s", color.CyanString("NamedInstance("), color.YellowString(fmt.Sprintf("\"%s\"", name)), color.GreenString(debugTypeString(provider)), color.CyanString(")")))
	}

	err := injector.bind(provider, name, false)
	if err != nil {
		injector.handleError(err)
	}

	return injector
}

// Reset deletes all the existing bindings from the injector instance.
func (injector *Injector) Reset() {
	if injector.isVerbose() {
		injector.logDebug(color.CyanString("Reset()"))
	}

	for k := range injector.bindings {
		delete(injector.bindings, k)
	}
}

// Call takes a function (receiver) with one or more arguments of the abstractions (interfaces).
// It invokes the function (receiver) and passes the related implementations.
func (injector *Injector) Call(function interface{}) {
	if injector.isVerbose() {
		injector.logDebug(fmt.Sprintf("%s%s%s", color.CyanString("Call("), color.GreenString(debugTypeString(function)), color.CyanString(")")))
	}

	err := injector.call(function)
	if err != nil {
		injector.handleError(err)
	}
}

func (injector *Injector) call(function interface{}) error {
	receiverType := reflect.TypeOf(function)
	if receiverType == nil {
		return injector.errorMiddleWare(fmt.Errorf("invalid function argument `%v`", function))
	}
	if receiverType.Kind() != reflect.Func {
		return injector.errorMiddleWare(fmt.Errorf("invalid function argument `%v` of type `%s`, argument must be a function", function, fullyQualifiedTypeString(receiverType)))
	}

	arguments, err := injector.arguments(function)
	if err != nil {
		return err
	}

	reflect.ValueOf(function).Call(arguments)

	return nil
}

// Resolve takes an abstraction (interface reference) and fills it with the related implementation.
func (injector *Injector) Resolve(abstraction interface{}) {
	if injector.isVerbose() {
		injector.logDebug(fmt.Sprintf("%s%s%s", color.CyanString("Resolve("), color.BlueString(debugNameString(abstraction)), color.CyanString(")")))
	}

	err := injector.resolve(abstraction, "")
	if err != nil {
		injector.handleError(err)
		return
	}
}

// NamedResolve resolves like the Resolve method but for named bindings.
func (injector *Injector) NamedResolve(abstraction interface{}, name string) {
	if injector.isVerbose() {
		injector.logDebug(fmt.Sprintf("%s%s%s", color.CyanString("NamedResolve("), color.BlueString(debugNameString(abstraction)), color.CyanString(")")))
	}

	err := injector.resolve(abstraction, name)
	if err != nil {
		injector.handleError(err)
		return
	}
}

func fullyQualifiedTypeString(t reflect.Type) string {
	path := t.PkgPath()
	if path == "" {
		return t.String()
	}
	return fmt.Sprintf("%value.%s", path, t.Name())
}

func (injector *Injector) resolve(abstraction interface{}, name string) error {
	receiverType := reflect.TypeOf(abstraction)
	if receiverType == nil {
		return injector.errorMiddleWare(fmt.Errorf("invalid abstraction argument `%+v`, ensure interface arguments are passed by reference (i.e Resolve(&arg))", abstraction))
	}

	if receiverType.Kind() != reflect.Ptr {
		return injector.errorMiddleWare(fmt.Errorf("invalid abstraction argument `%+v` of type `%s`, argument must be a struct or interface", abstraction, fullyQualifiedTypeString(receiverType)))
	}

	elem := receiverType.Elem()
	if elem.Kind() != reflect.Struct && elem.Kind() != reflect.Interface && elem.Kind() != reflect.Ptr {
		return injector.errorMiddleWare(fmt.Errorf("invalid abstraction argument `%+v` of type `%s`, argument must be a struct or interface", abstraction, fullyQualifiedTypeString(receiverType)))
	}

	concrete, exist := injector.bindings[elem][name]
	if !exist {
		concrete, exist = injector.bindings[receiverType][name]
		if !exist {
			return injector.errorMiddleWare(fmt.Errorf("no provider found for argument of type `%s`, ensure the type provided matches the return value of the provider", fullyQualifiedTypeString(elem)))
		}
		return injector.errorMiddleWare(fmt.Errorf("provider found for argument of type `%s`, but the argument was not passed by reference (i.e Resolve(&arg))", fullyQualifiedTypeString(elem)))
	}

	instance, err := concrete.resolve(injector, name, map[reflect.Type]map[string]interface{}{})
	if err != nil {
		return err
	}

	reflect.ValueOf(abstraction).Elem().Set(reflect.ValueOf(instance))
	return nil
}

// Fill takes a struct and resolves the fields with the tag `di:"inject"`
func (injector *Injector) Fill(structure interface{}) {
	if injector.isVerbose() {
		injector.logDebug(fmt.Sprintf("%s%s%s", color.CyanString("Fill("), color.BlueString(debugNameString(structure)), color.CyanString(")")))
	}

	err := injector.fill(structure, map[reflect.Type]map[string]interface{}{})
	if err != nil {
		injector.handleError(err)
		return
	}
}

func (injector *Injector) fill(structure interface{}, instantiated map[reflect.Type]map[string]interface{}) error {
	receiverType := reflect.TypeOf(structure)
	if receiverType == nil {
		return injector.errorMiddleWare(fmt.Errorf("invalid struct argument `%v`", structure))
	}

	if receiverType.Kind() != reflect.Ptr && receiverType.Elem().Kind() != reflect.Interface {
		return injector.errorMiddleWare(fmt.Errorf("argument of type `%s` is not a pointer or interface", fullyQualifiedTypeString(receiverType)))
	}

	// Allow passing structs by pointer values or pointer references i.e support both Fill(myPtr) and Fill(&myPtr)

	// Continue to unwrap the value until we have the underlying struct value
	value := reflect.ValueOf(structure)
	maxUnwrapAttempts := 4
	for i := 0; i < maxUnwrapAttempts; i++ {
		if value.Kind() == reflect.Ptr || value.Kind() == reflect.Interface {
			value = value.Elem()
		}
	}

	// If the underlying type is not a struct, error
	if value.Kind() != reflect.Struct {
		return injector.errorMiddleWare(fmt.Errorf("argument of type `%s` is not a struct", value.Type()))
	}

	if injector.isVerbose() {
		injector.incrementLoggerIndent()
		defer injector.decrementLoggerIndent()
	}

	for i := 0; i < value.NumField(); i++ {
		f := value.Field(i)

		t, exist := value.Type().Field(i).Tag.Lookup(tagName)
		if !exist {
			// field has no tag
			continue
		}

		var name string

		if t == injectByType {
			if injector.isVerbose() {
				injector.logDebug(fmt.Sprintf("%s: field `%s %s` by type", color.MagentaString(fillingPrefix), color.BlueString(value.Type().Field(i).Name), color.GreenString(fullyQualifiedTypeString(value.Type().Field(i).Type))))
			}
			name = ""
		} else if t == injectByName {
			if injector.isVerbose() {
				injector.logDebug(fmt.Sprintf("%s: field `%s %s` by name", color.MagentaString(fillingPrefix), color.BlueString(value.Type().Field(i).Name), color.GreenString(fullyQualifiedTypeString(value.Type().Field(i).Type))))
			}
			name = value.Type().Field(i).Name
		} else {
			return injector.errorMiddleWare(fmt.Errorf("field `%s` has an invalid struct tag `%s`", value.Type().Field(i).Name, t))
		}

		concrete, exist := injector.bindings[f.Type()][name]
		if !exist {
			return injector.errorMiddleWare(fmt.Errorf("cannot resolve field `%s %s`, no provider exists for type `%s` under name: `%s` ", value.Type().Field(i).Name, fullyQualifiedTypeString(value.Type().Field(i).Type), fullyQualifiedTypeString(value.Type().Field(i).Type), value.Type().Field(i).Name))
		}
		instance, err := concrete.resolve(injector, name, instantiated)
		if err != nil {
			injector.handleError(err)
		}

		if f.CanAddr() {
			ptr := reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr())).Elem()
			ptr.Set(reflect.ValueOf(instance))
		} else {
			if f.CanSet() {
				f.Set(reflect.ValueOf(instance))
			} else {
				return injector.errorMiddleWare(fmt.Errorf("field `%s %s` is not an addressible or settable field, must be a pointer or inteface type", value.Type().Field(i).Name, fullyQualifiedTypeString(value.Type().Field(i).Type)))
			}
		}
	}

	return nil

}
