# godi (Golang Dependency Injection Library)

## `Singletons` vs `Instances` providers

Singleton providers will be executed once and the resulting instance will be shared between all injections. These are ideal for stateless and/or threadsafe constructs. Singleton providers are evaluated _lazily_ which means the provider is not called until the moment of injection.

Instance providers will be executed for each injection and the resulting instances will not shared between injections. These are ideal for stateful or non-threadsafe constructs that should not be shared.

## `Resolve` example:

Here we register a singleton provider (called once and the value is shared) to resolve an interface to it's provided the concrete value.

*Note*: the return value of the provider and the argument type passed to `Resolve` must match.
*Note*: arguments to `Resolve` _must_ be passed by reference (including interfaces and pointers)

```go
injector := di.NewInjector() // or di.GlobalInjector

// register a "singleton" provider
injector.Singleton(func () MyServiceInterface {
    return &myServiceImpl{}
})

// resolve the interace to its concrete type
var myService MyServiceInterface
injector.Resolve(&myService)
```

## `NamedResolve` example:

Here we register a singleton provider (called once and the value is shared) under a particular key to resolve an interface to it's provided the concrete value.

*Note*: the return value of the provider and the argument type passed to `Resolve` must match.
*Note*: arguments to `NamedResolve` _must_ be passed by reference (including interfaces and pointers)

```go
// you can register providers under string keys
injector.NamedSingleton("mocked", func () MyServiceInterface {
    return &myMockedServiceImpl{}
})

// and resolve from these keys
var myMockedService MyServiceInterface
injector.NamedResolve("mocked", &myMockedService)
```

## `Fill` example:

Here we register an "instance" provider (called for each dependency, all values are unique) and use it to "fill" fields of an parent struct based on type.

*Note*: the return value of the provider and the tagged fields must match.
*Note*: structs may be passed as pointers or references of pointers.

```go

type NestedType {
    val int
}

type ValueType {
    val int
}

type MyStruct {
    nested *NestedType `di:"type"`
    value  ValueType   `di:"type"`
}

injector := di.NewInjector() // or di.GlobalInjector

// register an "instance" provider
injector.Instance(func () ValueType {
    return NestedType{
        val: 12,
    }
})
injector.Instance(func () *NestedType {
    return &NestedType{
        val: 42,
    }
})

// fill the fields by "type"
myStruct := &MyStruct{}
injector.Fill(myStruct)
fmt.Println(myStruct.nested.val) // "42"
fmt.Println(myStruct.value.val) // "12"
```

## `NamedFill` example:

Here we register two "instance" providers (called for each dependency, all values are unique) under unique keys and use it to "fill" fields of an parent struct based on field name.

*Note*: the return value of the provider and the tagged fields must match.
*Note*: structs may be passed as pointers or references of pointers.

```go

// we can can also fill by name:
type MyOtherStruct struct {
    a *NestedType `di:"type"`
    b *NestedType `di:"type"`
}

injector.NamedInstance("a", func () *NestedType {
    return &NestedType{
        val: 123,
    }
})

injector.NamedInstance("b", func () *NestedType {
    return &NestedType{
        val: 456,
    }
})

myOtherStruct := &MyOtherStruct{}
injector.Fill(&myOtherStruct) // passing the reference to the pointer is okay
fmt.Println(myOtherStruct.a.val) // "123"
fmt.Println(myOtherStruct.b.val) // "456"
```

## `Call` example:

You can invoke the injector to give you a concrete type for provided closure:

```go
injector := di.NewInjector() // or di.GlobalInjector

// register a "singleton" provider
injector.Singleton(func () MyServiceInterface {
    return &myServiceImpl{}
})

injector.Call(func (svc MyServiceInterface) {
    svc.DoWhatever()
})
```

## Thread safety:

Do not register providers via `Singleton`, `NamedSingleton`, `Instance`, and `NamedInstace` while at the same time calling `Fill` or `Resolve` or `Call` from a separate goroutine.

The `Fill` or `Resolve` or `Call` methods _read_ from a map of bindings, and the `Singleton`, `NamedSingleton`, `Instance`, and `NamedInstace` methods _write_ to that map of bindings.

This shouldn't every really happen, since you would always want to define all your providers sequentially at the startup of an application.

Everything else is threadsafe and can be called across goroutines.

## Debugging:

Verbose debug logs can be enabled / disabled and can be useful when debugging injection failures.

Ex.
```go
func NewSomething(injector *di.Injector) *Something {
    injector.EnableDebugLogging()            // <-- add this in if you are dealing
    defer injector.DisableDebugLogging()     //     with some tricky di errors

    s := &Something{}
    injector.Fill(s)
    return s
}
```

As the logs are _very_ verbose, it is recommended that the enable / disable calls be scoped as tightly to the source of error as possible.

## Handling errors:

By default, if there is an internal error encountered by the `di` package, it will panic. To capture and handle errors you can provide an error handler:

```go

injector := di.NewInjector()
injector.SetErrorHandler(func (err error) {
    // log the error
})
```

A injection error should be considered "fatal" and should be resolved in development / QA. They are not events to be handeld gracefully in production.

## Future Work:

## Auto-filling returned values and removing manual injection.

One option is to remove the need to call `Fill` at all. Instead we could make it a private method, and automatically call it on every return value from a provider.

Pros:
    - less boilerplate calls to `Fill` in providers / constructors
    - no need to pass `*di.Injector` around to fill things
Cons:
    - More implicit "magical" behavior
    - Can no longer support returning "value" types, since internally we cannot discern the type of a `&interface{}`

## The introduction of generics could provide a much cleaner interface for instantiating types:

Both `Resolve` and `Fill` could be replaced with a single:

```go
value := injector.Get[SomeType]()
```

`di.Injector.Get` would automatically `Resolve` any interface type registered via a provider, and would instantiate and `Fill` any struct / pointer type.

## Acknowledgements

A huge thank you to [Kevin Birke](https://github.com/kbirk) for the design and contributions to this repo.