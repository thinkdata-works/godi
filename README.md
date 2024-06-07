# godi (Golang Dependency Injection Library)

A simple dependency injection library inspired by [golobby](https://github.com/golobby/container)

## Requirements

`godi` requires go version 1.22 because it uses the newly added `reflect.TypeFor[T]`.

## `Get` example:

Here we register a singleton provider (called once and the value is shared) to resolve an interface to it's provided the concrete value.

*Note*: the return value of the provider and the argument type passed to `Get` must match exactly. If the provider returns an pointer to a struct rather than an interface, even if that struct fulfils the interface, the lookup for the bindings will fail.

```go
injector := di.NewInjector() // or di.GlobalInjector

// register a "singleton" provider
injector.Singleton(func () MyServiceInterface {
    return &myServiceImpl{}
})

// resolve the interface to its concrete type
myService := di.Get[MyServiceInterface](injector)

// you can register providers under string keys
injector.NamedSingleton("a", func () MyServiceInterface {
    return &ServiceImplA{}
})
injector.NamedSingleton("b", func () MyServiceInterface {
    return &ServiceImplB{}
})

// and resolve from these keys
aService := di.Get[MyServiceInterface](injector, "a")
bService := di.Get[MyServiceInterface](injector, "b")
```

## `Resolve` / `NamedResolve` examples:

Here we register a singleton provider (called once and the value is shared) to resolve an interface to it's provided the concrete value.

*Note*: the return value of the provider and the argument type passed to `Resolve` must match.
*Note*: arguments to `Resolve` / `NamedResolve` _must_ be passed by reference (including interfaces and pointers)

```go
injector := di.NewInjector() // or di.GlobalInjector

// register a "singleton" provider
injector.Singleton(func () MyServiceInterface {
    return &myServiceImpl{}
})

// resolve the interface to its concrete type
var myService MyServiceInterface
injector.Resolve(&myService)

// you can register providers under string keys
injector.NamedSingleton("a", func () MyServiceInterface {
    return &ServiceImplA{}
})
injector.NamedSingleton("b", func () MyServiceInterface {
    return &ServiceImplB{}
})

// and resolve from these keys
var aService, bService MyServiceInterface
injector.NamedResolve("a", &aService)
injector.NamedResolve("b", &bService)
```

## `Fill` examples:

Here we register an "instance" provider (called for each dependency, all values are unique) and use it to "fill" fields of an parent struct based on type.

*Note*: the return value of the provider and the tagged fields must match.
*Note*: structs may be passed as pointers or references of pointers.

```go

type NestedType {
    val int
}

type AnotherType {
    val int
}

type MyStruct {
    nested *NestedType `di:"type"`
    value  *AnotherType*   `di:"type"`
}

injector := di.NewInjector() // or di.GlobalInjector

// register an "instance" provider
injector.Instance(func () *AnotherType* {
    return AnotherType{
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


// we can can also fill by name:
type MyOtherStruct struct {
    a *NestedType `di:"name"`
    b *NestedType `di:"name"`
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

## `Singletons` vs `Instances` providers

Singleton providers will be executed once and the resulting instance will be shared between all injections. These are ideal for stateless and/or threadsafe constructs. Singleton providers are evaluated _lazily_ which means the provider is not called until the moment of injection.

Instance providers will be executed for each injection and the resulting instances will not shared between injections. These are ideal for stateful or non-threadsafe constructs that should not be shared.

## Circular dependencies

`godi` handles circular dependencies for both singleton and instance methods. For singletons the cyclic properties will all point to the same resolved singletonvalues. For cyclic instance instantiation, instances will point to the same resolved values _within the injection call_.

```go
type A struct {
    B *B `di:"type"`
}

type B struct {
    A *A `di:"type"
}

injector.Singleton(func () *A {
    return &A{}
})
injector.Instance(func () *B {
    return &B{}
})

a := di.Get[*A](injector)
fmt.Println(a.B.A.B.A.B != nil) // true
```

## Thread safety:

Do not register providers via `Singleton`, `NamedSingleton`, `Instance`, and `NamedInstace` while at the same time calling `Fill` or `Resolve` or `Call` from a separate goroutine.

The `Fill` or `Resolve` or `Call` methods _read_ from a map of bindings, and the `Singleton`, `NamedSingleton`, `Instance`, and `NamedInstace` methods _write_ to that map of bindings.

This shouldn't every really happen, since you would always want to define all your providers sequentially at the startup of an application before use.

Everything else is threadsafe and can be called across goroutines.

## Debugging:

Verbose debug logs can be enabled / disabled and can be useful when debugging injection failures.

Ex.
```go
injector.EnableDebugLogging()            // <-- add this in if you are dealing with some tricky di errors
defer injector.DisableDebugLogging()

s:= injector.Get[*Something]()
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

## Acknowledgements

- A huge thank you to [Kevin Birk](https://github.com/kbirk) for the design and contributions to this repo.
- Thanks to [golobby](https://github.com/golobby/container) for inspiring this repo and serving as a code reference for how to use the `reflect` package to accomplish sane dependency injection in go.
