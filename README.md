# ftl : function transformer library

Functional sequencing combinators written in Go. A continuation of my explorations in functional Go [here](https://github.com/nytopop/errsel), specialized to a small set of requirements commonly found in real code.

Inspired by [Control.Monad](https://hackage.haskell.org/package/base-4.11.0.0/docs/Control-Monad.html). Unfortunately, only a small portion of its functionality can be implemented sanely in Go.

This library essentially provides a DSL for gluing together and modifying the runtime behavior of the typed functions:

```Go
// Predicate handles errors.
type Predicate func(error) bool

// Closure is a function that might fail.
type Closure func() error

// Tasklet is an interruptible Closure.
type Tasklet func(context.Context) error

// Statelet is a stateful Closure.
type Statelet func(StateLoader) error

// Routine is a stateful, interruptible Closure.
type Routine func(context.Context, StateLoader) error
```

Some subpackages utilizing the primitives in `ftl` are also provided.
