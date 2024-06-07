package di

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type Shape interface {
	SetArea(int)
	GetArea() int
}

type Circle struct {
	a int
}

func (c *Circle) SetArea(a int) {
	c.a = a
}

func (c Circle) GetArea() int {
	return c.a
}

type TypeC struct {
	val int
}

type TypeB struct {
	c *TypeC `di:"type"`
}

type TypeA struct {
	b *TypeB `di:"type"`
}

func (t *TypeA) DoSomething() {
	t.b.c.val *= 2
}

type ValueC struct {
	val int
}

type ValueB struct {
	c ValueC `di:"type"`
}

type ValueA struct {
	b ValueB `di:"type"`
}

type Database interface {
	Connect() bool
}

type MySQL struct{}

func (m MySQL) Connect() bool {
	return true
}

func TestInjector_Singleton(t *testing.T) {
	var injector = NewInjector()
	injector.verbose = 1
	injector.SetErrorHandler(func(err error) {
		assert.NoError(t, err)
	})
	injector.Singleton(func() (Shape, error) {
		return &Circle{a: 13}, nil
	})
	injector.Singleton(func() *TypeC {
		return &TypeC{}
	})
	injector.Singleton(func() *TypeB {
		return &TypeB{}
	})
	injector.Singleton(func() *TypeA {
		a := &TypeA{}
		injector.Fill(&a)
		return a
	})
	injector.Call(func(s1 Shape) {
		s1.SetArea(42)
	})
	injector.Call(func(s2 Shape) {
		a := s2.GetArea()
		assert.Equal(t, 42, a)
	})

	// test that they are instantiated lazily
	wasCalled := false
	injector.Singleton(func() Database {
		wasCalled = true
		return &MySQL{}
	})
	assert.Equal(t, false, wasCalled)
	injector.Call(func(db Database) {
		assert.Equal(t, true, wasCalled)
	})

	shape := Get[Shape](injector)
	assert.NotNil(t, shape)
	assert.Equal(t, 42, shape.GetArea())

	ta := Get[*TypeA](injector)
	assert.NotNil(t, ta)
	assert.NotNil(t, ta.b)

	// should handle nil return values without a panic
	injector.SetErrorHandler(func(err error) {
		fmt.Println(err.Error())
		assert.Error(t, err)
	})
	Get[TypeA](injector)
}

func TestInjector_Singleton_Fail(t *testing.T) {
	var injector = NewInjector()
	errorCount := 0
	injector.SetErrorHandler(func(err error) {
		assert.Error(t, err)
		errorCount++
	})
	injector.Singleton(func() {}) // no return values should produce an error
	assert.Equal(t, 1, errorCount)
	injector.Singleton(func() TypeC { return TypeC{} }) // must return a reference type
	assert.Equal(t, 2, errorCount)
	injector.Singleton(func() string { return "test-string" }) // must return a reference type
	assert.Equal(t, 3, errorCount)
	injector.Singleton("STRING!") // must receive a function
	assert.Equal(t, 4, errorCount)
	injector.Singleton(func(i int) *TypeC { return &TypeC{} }) // must receive a function with no args
	assert.Equal(t, 5, errorCount)
	injector.Singleton(func() *TypeC { return nil }) // must not return nil
	injector.Call(func(t *TypeC) {})
	assert.Equal(t, 6, errorCount)
	injector.Singleton(func() (*TypeC, error) { return nil, fmt.Errorf("err") }) // provider returned an error
	injector.Call(func(t *TypeC) {})
	assert.Equal(t, 7, errorCount)
}
func TestInjector_Singleton_Fill(t *testing.T) {
	var injector = NewInjector()
	injector.SetErrorHandler(func(err error) {
		assert.NoError(t, err)
	})
	injector.Singleton(func() Shape {
		return &Circle{a: 13}
	})

	var sh Shape
	injector.Resolve(&sh)
	assert.Equal(t, 13, sh.GetArea())
}

type John struct {
	Val int
}

type Alice struct {
	Bob  *Bob  `di:"type"`
	John *John `di:"type"`
}

type Bob struct {
	Alice *Alice `di:"type"`
	John  *John  `di:"type"`
}

func TestInjector_Circular_Deps_Singleton(t *testing.T) {
	var injector = NewInjector()
	injector.SetErrorHandler(func(err error) {
		assert.NoError(t, err)
	})

	injector.Singleton(func() *Alice {
		return &Alice{}
	})
	injector.Singleton(func() *Bob {
		return &Bob{}
	})
	injector.Singleton(func() *John {
		return &John{Val: 42}
	})

	alice := Get[*Alice](injector)
	assert.NotNil(t, alice)
	assert.NotNil(t, alice.John)
	assert.Equal(t, 42, alice.John.Val)
	assert.NotNil(t, alice.Bob)
	assert.NotNil(t, alice.Bob.John)
	assert.Equal(t, 42, alice.Bob.John.Val)
	assert.NotNil(t, alice.Bob.Alice)
	assert.NotNil(t, alice.Bob.Alice.John)
	assert.Equal(t, 42, alice.Bob.Alice.John.Val)
	assert.NotNil(t, alice.Bob.Alice.Bob)
	assert.NotNil(t, alice.Bob.Alice.Bob.John)
	assert.Equal(t, 42, alice.Bob.Alice.Bob.John.Val)
	assert.NotNil(t, alice.Bob.Alice.Bob.Alice)
	assert.NotNil(t, alice.Bob.Alice.Bob.Alice.John)
	assert.Equal(t, 42, alice.Bob.Alice.Bob.Alice.John.Val)
	// to infinity and beyond
}

func TestInjector_Circular_Deps_Singleton_Call(t *testing.T) {
	var injector = NewInjector()
	injector.SetErrorHandler(func(err error) {
		assert.NoError(t, err)
	})

	injector.Singleton(func() *Alice {
		return &Alice{}
	})
	injector.Singleton(func() *Bob {
		return &Bob{}
	})
	injector.Singleton(func() *John {
		return &John{Val: 42}
	})

	injector.Call(func(alice *Alice) {
		assert.NotNil(t, alice)
		assert.NotNil(t, alice.John)
		assert.Equal(t, 42, alice.John.Val)
		assert.NotNil(t, alice.Bob)
		assert.NotNil(t, alice.Bob.John)
		assert.Equal(t, 42, alice.Bob.John.Val)
		assert.NotNil(t, alice.Bob.Alice)
		assert.NotNil(t, alice.Bob.Alice.John)
		assert.Equal(t, 42, alice.Bob.Alice.John.Val)
		assert.NotNil(t, alice.Bob.Alice.Bob)
		assert.NotNil(t, alice.Bob.Alice.Bob.John)
		assert.Equal(t, 42, alice.Bob.Alice.Bob.John.Val)
		assert.NotNil(t, alice.Bob.Alice.Bob.Alice)
		assert.NotNil(t, alice.Bob.Alice.Bob.Alice.John)
		assert.Equal(t, 42, alice.Bob.Alice.Bob.Alice.John.Val)
		// to infinity and beyond
	})
}

func TestInjector_Circular_Deps_Instance(t *testing.T) {
	var injector = NewInjector()
	injector.SetErrorHandler(func(err error) {
		assert.NoError(t, err)
	})

	injector.Instance(func() *Alice {
		return &Alice{}
	})
	injector.Instance(func() *Bob {
		return &Bob{}
	})
	injector.Instance(func() *John {
		return &John{Val: 42}
	})

	alice := Get[*Alice](injector)
	assert.NotNil(t, alice)
	assert.NotNil(t, alice.John)
	assert.Equal(t, 42, alice.John.Val)
	assert.NotNil(t, alice.Bob)
	assert.NotNil(t, alice.Bob.John)
	assert.Equal(t, 42, alice.Bob.John.Val)
	assert.NotNil(t, alice.Bob.Alice)
	assert.NotNil(t, alice.Bob.Alice.John)
	assert.Equal(t, 42, alice.Bob.Alice.John.Val)
	assert.NotNil(t, alice.Bob.Alice.Bob)
	assert.NotNil(t, alice.Bob.Alice.Bob.John)
	assert.Equal(t, 42, alice.Bob.Alice.Bob.John.Val)
	assert.NotNil(t, alice.Bob.Alice.Bob.Alice)
	assert.NotNil(t, alice.Bob.Alice.Bob.Alice.John)
	assert.Equal(t, 42, alice.Bob.Alice.Bob.Alice.John.Val)
	// to infinity and beyond
}

func TestInjector_Circular_Deps_Instance_Call(t *testing.T) {
	var injector = NewInjector()
	injector.SetErrorHandler(func(err error) {
		assert.NoError(t, err)
	})

	injector.Instance(func() *Alice {
		return &Alice{}
	})
	injector.Instance(func() *Bob {
		return &Bob{}
	})
	injector.Instance(func() *John {
		return &John{Val: 42}
	})

	injector.Call(func(alice *Alice) {
		assert.NotNil(t, alice)
		assert.NotNil(t, alice.John)
		assert.Equal(t, 42, alice.John.Val)
		assert.NotNil(t, alice.Bob)
		assert.NotNil(t, alice.Bob.John)
		assert.Equal(t, 42, alice.Bob.John.Val)
		assert.NotNil(t, alice.Bob.Alice)
		assert.NotNil(t, alice.Bob.Alice.John)
		assert.Equal(t, 42, alice.Bob.Alice.John.Val)
		assert.NotNil(t, alice.Bob.Alice.Bob)
		assert.NotNil(t, alice.Bob.Alice.Bob.John)
		assert.Equal(t, 42, alice.Bob.Alice.Bob.John.Val)
		assert.NotNil(t, alice.Bob.Alice.Bob.Alice)
		assert.NotNil(t, alice.Bob.Alice.Bob.Alice.John)
		assert.Equal(t, 42, alice.Bob.Alice.Bob.Alice.John.Val)
		// to infinity and beyond
	})
}

func TestInjector_Mix_Singleton_Instance_Recursive(t *testing.T) {
	var injector = NewInjector()
	injector.SetErrorHandler(func(err error) {
		assert.NoError(t, err)
	})

	injector.Singleton(func() *Alice {
		return &Alice{}
	})
	injector.Instance(func() *Bob {
		return &Bob{}
	})
	injector.Singleton(func() *John {
		return &John{Val: 42}
	})

	alice := Get[*Alice](injector)
	assert.NotNil(t, alice)
	assert.NotNil(t, alice.John)
	assert.Equal(t, 42, alice.John.Val)
	assert.NotNil(t, alice.Bob)
	assert.NotNil(t, alice.Bob.John)
	assert.Equal(t, 42, alice.Bob.John.Val)
	assert.NotNil(t, alice.Bob.Alice)
	assert.NotNil(t, alice.Bob.Alice.John)
	assert.Equal(t, 42, alice.Bob.Alice.John.Val)
	assert.NotNil(t, alice.Bob.Alice.Bob)
	assert.NotNil(t, alice.Bob.Alice.Bob.John)
	assert.Equal(t, 42, alice.Bob.Alice.Bob.John.Val)
	assert.NotNil(t, alice.Bob.Alice.Bob.Alice)
	assert.NotNil(t, alice.Bob.Alice.Bob.Alice.John)
	assert.Equal(t, 42, alice.Bob.Alice.Bob.Alice.John.Val)
	// to infinity and beyond
}

type Partner struct {
	Name   string
	Robert *Robert `di:"type"`
}

type Robert struct {
	Dale  *Partner `di:"name"`
	Peter *Partner `di:"name"`
}

func TestInjector_NamedSingletons_Resursive(t *testing.T) {
	var injector = NewInjector()
	injector.SetErrorHandler(func(err error) {
		assert.NoError(t, err)
	})
	injector.NamedSingleton("Dale", func() *Partner {
		return &Partner{Name: "dale"}
	})
	injector.NamedSingleton("Peter", func() *Partner {
		return &Partner{Name: "peter"}
	})
	injector.Singleton(func() *Robert {
		return &Robert{}
	})

	robert := Get[*Robert](injector)
	assert.NotNil(t, robert)
	assert.NotNil(t, robert.Dale)
	assert.NotNil(t, robert.Peter)
	assert.Equal(t, "dale", robert.Dale.Name)
	assert.Equal(t, "peter", robert.Peter.Name)
	assert.NotNil(t, robert.Dale.Robert)
	assert.NotNil(t, robert.Peter.Robert)
	assert.NotNil(t, robert.Dale.Robert.Peter)
	assert.NotNil(t, robert.Dale.Robert.Dale)
	assert.NotNil(t, robert.Peter.Robert.Peter)
	assert.NotNil(t, robert.Dale.Robert.Dale)
}

func TestInjector_NamedInstances_Resursive(t *testing.T) {
	var injector = NewInjector()
	injector.SetErrorHandler(func(err error) {
		assert.NoError(t, err)
	})
	injector.NamedInstance("Dale", func() *Partner {
		return &Partner{Name: "dale"}
	})
	injector.NamedInstance("Peter", func() *Partner {
		return &Partner{Name: "peter"}
	})
	injector.Singleton(func() *Robert {
		return &Robert{}
	})

	robert := Get[*Robert](injector)
	assert.NotNil(t, robert)
	assert.NotNil(t, robert.Dale)
	assert.NotNil(t, robert.Peter)
	assert.Equal(t, "dale", robert.Dale.Name)
	assert.Equal(t, "peter", robert.Peter.Name)
	assert.NotNil(t, robert.Dale.Robert)
	assert.NotNil(t, robert.Peter.Robert)
	assert.NotNil(t, robert.Dale.Robert.Peter)
	assert.NotNil(t, robert.Dale.Robert.Dale)
	assert.NotNil(t, robert.Peter.Robert.Peter)
	assert.NotNil(t, robert.Dale.Robert.Dale)
}

func TestInjector_NamedInstances_NamedSingleton_Mix_Resursive(t *testing.T) {
	var injector = NewInjector()
	injector.SetErrorHandler(func(err error) {
		assert.NoError(t, err)
	})
	injector.NamedInstance("Dale", func() *Partner {
		return &Partner{Name: "dale"}
	})
	injector.NamedSingleton("Peter", func() *Partner {
		return &Partner{Name: "peter"}
	})
	injector.Singleton(func() *Robert {
		return &Robert{}
	})

	robert := Get[*Robert](injector)
	assert.NotNil(t, robert)
	assert.NotNil(t, robert.Dale)
	assert.NotNil(t, robert.Peter)
	assert.Equal(t, "dale", robert.Dale.Name)
	assert.Equal(t, "peter", robert.Peter.Name)
	assert.NotNil(t, robert.Dale.Robert)
	assert.NotNil(t, robert.Peter.Robert)
	assert.NotNil(t, robert.Dale.Robert.Peter)
	assert.NotNil(t, robert.Dale.Robert.Dale)
	assert.NotNil(t, robert.Peter.Robert.Peter)
	assert.NotNil(t, robert.Dale.Robert.Dale)
}

func TestInjector_NamedSingleton(t *testing.T) {
	var injector = NewInjector()
	injector.SetErrorHandler(func(err error) {
		assert.NoError(t, err)
	})
	injector.NamedSingleton("theCircle", func() Shape {
		return &Circle{a: 13}
	})

	var sh Shape
	injector.NamedResolve(&sh, "theCircle")
	assert.Equal(t, 13, sh.GetArea())
}

type IParent interface {
	GetA() *A
}

type Parent struct {
	A *A `di:"type"`
}

func (a *Parent) GetA() *A {
	return a.A
}

type A struct {
	B *B `di:"type"`
}

type B struct {
	C *C `di:"type"`
}

type C struct {
	Val int
}

func TestInjector_FillRecursively(t *testing.T) {
	var injector = NewInjector()
	injector.SetErrorHandler(func(err error) {
		assert.NoError(t, err)
	})
	injector.Singleton(func() IParent {
		return &Parent{}
	})
	injector.Singleton(func() *A {
		return &A{}
	})
	injector.Singleton(func() *B {
		return &B{}
	})
	injector.Singleton(func() *C {
		return &C{
			Val: 123,
		}
	})

	p := Get[IParent](injector)
	assert.NotNil(t, p)
	assert.NotNil(t, p.GetA())
	assert.NotNil(t, p.GetA().B)
	assert.NotNil(t, p.GetA().B.C)
	assert.Equal(t, 123, p.GetA().B.C.Val)

	a := Get[*A](injector)
	assert.NotNil(t, a)
	assert.NotNil(t, a.B)
	assert.NotNil(t, a.B.C)
	assert.Equal(t, 123, a.B.C.Val)
}

func TestInjector_Instance(t *testing.T) {
	var injector = NewInjector()
	injector.SetErrorHandler(func(err error) {
		assert.NoError(t, err)
	})
	injector.Instance(func() Shape {
		return &Circle{a: 42}
	})
	injector.Call(func(s1 Shape) {
		s1.SetArea(13)
	})
	injector.Call(func(s2 Shape) {
		a := s2.GetArea()
		assert.Equal(t, 42, a)
	})

	injector.Instance(func() *TypeC {
		return &TypeC{val: 42}
	})
	injector.Instance(func() *TypeB {
		return &TypeB{}
	})
	injector.Instance(func() *TypeA {
		return &TypeA{}
	})
	injector.Call(func(a *TypeA) {
		assert.NotNil(t, a)
		assert.NotNil(t, a.b)
		assert.NotNil(t, a.b.c)
		assert.EqualValues(t, a.b.c.val, 42)
		a.DoSomething()
	})

	injector.Reset()
	injector.Instance(func() *TypeC {
		return &TypeC{val: 42}
	})
	injector.Instance(func() *TypeB {
		return &TypeB{}
	})
	injector.Instance(func() (*TypeA, error) {
		return &TypeA{}, nil
	})
	a := &TypeA{}
	injector.Resolve(&a)
	assert.NotNil(t, a.b)
	assert.NotNil(t, a.b.c)
	assert.EqualValues(t, a.b.c.val, 42)
}

func TestInjector_Instance_With_Resolve_That_Returns_Error(t *testing.T) {
	var injector = NewInjector()
	injector.Instance(func() (Shape, error) {
		return nil, errors.New("test error")
	})
	injector.SetErrorHandler(func(err error) {
		assert.Error(t, err)
	})
	var s Shape
	injector.Resolve(&s)
	assert.Nil(t, s)

	injector.Call(func(s Shape) {
		assert.Fail(t, "this should not execute")
	})
}

func TestInjector_Instance_With_Resolve_With_Invalid_Signature_It_Should_Fail(t *testing.T) {
	var injector = NewInjector()
	injector.SetErrorHandler(func(err error) {
		assert.Error(t, err)
	})
	injector.Instance(func() (Shape, Database, error) {
		return nil, nil, nil
	})
	var s Shape
	injector.Resolve(&s)
	assert.Nil(t, s)
}

func TestInjector_NamedInstance(t *testing.T) {
	var injector = NewInjector()
	injector.SetErrorHandler(func(err error) {
		assert.NoError(t, err)
	})
	injector.NamedInstance("theCircle", func() Shape {
		return &Circle{a: 13}
	})

	var sh Shape
	injector.NamedResolve(&sh, "theCircle")
	assert.Equal(t, 13, sh.GetArea())
}

func TestInjector_Call_With_Multiple_Resolving(t *testing.T) {
	var injector = NewInjector()
	injector.SetErrorHandler(func(err error) {
		assert.NoError(t, err)
	})
	injector.Singleton(func() Shape {
		return &Circle{a: 5}
	})
	injector.Singleton(func() Database {
		return &MySQL{}
	})
	injector.Call(func(s Shape, m Database) {
		if _, ok := s.(*Circle); !ok {
			t.Error("Expected Circle")
		}

		if _, ok := m.(*MySQL); !ok {
			t.Error("Expected MySQL")
		}
	})
}

func TestInjector_Call_With_Unsupported_Receiver_It_Should_Fail(t *testing.T) {
	var injector = NewInjector()
	injector.SetErrorHandler(func(err error) {
		assert.Error(t, err)
	})
	injector.Call("STRING!")
}

func TestInjector_Call_With_Second_UnBounded_Argument(t *testing.T) {
	var injector = NewInjector()
	injector.SetErrorHandler(func(err error) {
		assert.NoError(t, err)
	})
	injector.Singleton(func() Shape {
		return &Circle{}
	})
	injector.SetErrorHandler(func(err error) {
		assert.Error(t, err)
	})
	injector.Call(func(s Shape, d Database) {})
}

func TestInjector_Resolve_With_Reference_As_Resolver(t *testing.T) {
	var injector = NewInjector()
	injector.SetErrorHandler(func(err error) {
		assert.NoError(t, err)
	})
	injector.Singleton(func() Shape {
		return &Circle{a: 5}
	})

	injector.Singleton(func() Database {
		return &MySQL{}
	})

	var (
		s Shape
		d Database
	)

	injector.Resolve(&s)
	if _, ok := s.(*Circle); !ok {
		t.Error("Expected Circle")
	}

	injector.Resolve(&d)
	if _, ok := d.(*MySQL); !ok {
		t.Error("Expected MySQL")
	}
}

func TestInjector_Resolve_With_Unsupported_Receiver_It_Should_Fail(t *testing.T) {
	var injector = NewInjector()
	injector.SetErrorHandler(func(err error) {
		assert.Error(t, err)
	})
	injector.Resolve("STRING!")

	str := "STRING!"
	injector.Resolve(&str)
}

func TestInjector_Resolve_With_NonReference_Receiver_It_Should_Fail(t *testing.T) {
	var injector = NewInjector()
	injector.SetErrorHandler(func(err error) {
		assert.NoError(t, err)
	})

	injector.Singleton(func() Shape {
		return &Circle{a: 5}
	})

	injector.Singleton(func() *Circle {
		return &Circle{a: 5}
	})

	injector.SetErrorHandler(func(err error) {
		assert.Error(t, err)
	})

	var s Shape
	injector.Resolve(s)
	assert.Nil(t, s)

	var c *Circle
	injector.Resolve(c)
	assert.Nil(t, c)
}

func TestInjector_Resolve_With_UnBounded_Reference_It_Should_Fail(t *testing.T) {
	var injector = NewInjector()
	injector.SetErrorHandler(func(err error) {
		assert.Error(t, err)
	})
	var s Shape
	injector.Resolve(&s)
}

type SomeStruct struct {
	S Shape    `di:"type"`
	D Database `di:"type"`
	C Shape    `di:"name"`
	X string
}

func TestInjector_Fill_With_Struct_Pointer(t *testing.T) {
	var injector = NewInjector()
	injector.SetErrorHandler(func(err error) {
		assert.NoError(t, err)
	})
	injector.Singleton(func() Shape {
		return &Circle{a: 5}
	})

	injector.NamedSingleton("C", func() Shape {
		return &Circle{a: 5}
	})

	injector.Singleton(func() Database {
		return &MySQL{}
	})

	injector.Singleton(func() *SomeStruct {
		return &SomeStruct{}
	})

	myApp := struct {
		S Shape    `di:"type"`
		D Database `di:"type"`
		C Shape    `di:"name"`
		X string
	}{}

	injector.Fill(&myApp)

	assert.IsType(t, &Circle{}, myApp.S)
	assert.IsType(t, &MySQL{}, myApp.D)

	someStruct := Get[*SomeStruct](injector)
	assert.IsType(t, &Circle{}, someStruct.S)
	assert.IsType(t, &MySQL{}, someStruct.D)
	assert.IsType(t, &Circle{}, someStruct.C)
	assert.Equal(t, 5, someStruct.C.GetArea())

}

func TestInjector_Register_With_Struct_Value_Should_Error(t *testing.T) {
	var injector = NewInjector()
	injector.SetErrorHandler(func(err error) {
		assert.Error(t, err)
	})
	injector.Instance(func() ValueC {
		return ValueC{val: 5}
	})
	injector.Instance(func() ValueB {
		return ValueB{}
	})
	injector.Instance(func() ValueA {
		return ValueA{}
	})
}

func TestInjector_Fill_Unexported_With_Struct_Pointer(t *testing.T) {
	var injector = NewInjector()
	injector.SetErrorHandler(func(err error) {
		assert.NoError(t, err)
	})
	injector.Singleton(func() Shape {
		return &Circle{a: 5}
	})

	injector.Singleton(func() Database {
		return &MySQL{}
	})

	myApp := struct {
		s Shape    `di:"type"`
		d Database `di:"type"`
		y int
	}{}

	injector.Fill(&myApp)

	assert.IsType(t, &Circle{}, myApp.s)
	assert.IsType(t, &MySQL{}, myApp.d)

	injector.Instance(func() *TypeC {
		return &TypeC{val: 42}
	})
	injector.Instance(func() *TypeB {
		return &TypeB{}
	})
	injector.Instance(func() *TypeA {
		return &TypeA{}
	})

	// Should accept pointer
	a1 := &TypeA{}
	injector.Fill(a1)
	assert.NotNil(t, a1.b)
	assert.NotNil(t, a1.b.c)
	assert.EqualValues(t, a1.b.c.val, 42)

	// And should accept reference to the pointer
	a2 := &TypeA{}
	injector.Fill(&a2)
	assert.NotNil(t, a2.b)
	assert.NotNil(t, a2.b.c)
	assert.EqualValues(t, a2.b.c.val, 42)
}

func TestInjector_Fill_With_Invalid_Field_It_Should_Fail(t *testing.T) {
	var injector = NewInjector()
	injector.SetErrorHandler(func(err error) {
		assert.NoError(t, err)
	})
	injector.NamedSingleton("C", func() Shape {
		return &Circle{a: 5}
	})

	type App struct {
		S string `di:"name"`
	}

	myApp := App{}
	injector.SetErrorHandler(func(err error) {
		assert.Error(t, err)
	})
	injector.Fill(&myApp)
	assert.EqualValues(t, myApp.S, "")
}

func TestInjector_Fill_With_Invalid_Tag_It_Should_Fail(t *testing.T) {
	var injector = NewInjector()
	injector.SetErrorHandler(func(err error) {
		assert.NoError(t, err)
	})

	injector.NamedSingleton("C", func() Shape {
		return &Circle{a: 5}
	})
	type App struct {
		S string `di:"invalid"`
	}

	myApp := App{}

	injector.SetErrorHandler(func(err error) {
		assert.Error(t, err)
	})
	injector.Fill(&myApp)
	assert.EqualValues(t, myApp.S, "")
}

func TestInjector_Fill_With_Invalid_Field_Name_It_Should_Fail(t *testing.T) {
	var injector = NewInjector()
	injector.SetErrorHandler(func(err error) {
		assert.NoError(t, err)
	})
	injector.NamedInstance("C", func() *TypeC {
		return &TypeC{val: 42}
	})
	type App struct {
		S TypeC `di:"name"`
	}

	myApp := App{}
	injector.SetErrorHandler(func(err error) {
		assert.Error(t, err)
	})
	injector.Fill(&myApp)
	assert.EqualValues(t, myApp.S.val, 0)
}

func TestInjector_Fill_With_Invalid_Struct_It_Should_Fail(t *testing.T) {
	var injector = NewInjector()
	injector.SetErrorHandler(func(err error) {
		assert.NoError(t, err)
	})
	injector.Singleton(func() Shape {
		return &Circle{a: 5}
	})
	injector.SetErrorHandler(func(err error) {
		assert.Error(t, err)
	})
	invalidStruct := 0
	injector.Fill(&invalidStruct)
	assert.EqualValues(t, invalidStruct, 0)
}

func TestInjector_Fill_With_Invalid_Pointer_It_Should_Fail(t *testing.T) {
	var injector = NewInjector()
	injector.SetErrorHandler(func(err error) {
		assert.NoError(t, err)
	})
	injector.Singleton(func() Shape {
		return &Circle{a: 5}
	})
	injector.SetErrorHandler(func(err error) {
		assert.Error(t, err)
	})

	var s Shape
	injector.Fill(s)
	assert.Nil(t, s)
}

func TestInjector_Fill_With_With_No_Tags_Should_Fail(t *testing.T) {
	var injector = NewInjector()
	injector.SetErrorHandler(func(err error) {
		assert.NoError(t, err)
	})
	injector.SetErrorHandler(func(err error) {
		assert.Error(t, err)
	})

	c := &Circle{}
	injector.Fill(&c)
	assert.EqualValues(t, c.a, 0)
}

func TestInjector_Reset(t *testing.T) {
	var injector = NewInjector()
	injector.SetErrorHandler(func(err error) {
		assert.NoError(t, err)
	})

	injector.Singleton(func() Shape {
		return &Circle{a: 5}
	})
	injector.Reset()

	injector.SetErrorHandler(func(err error) {
		assert.Error(t, err)
	})

	var s Shape
	injector.Fill(s)
}
