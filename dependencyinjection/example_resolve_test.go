package dependencyinjection_test

import (
	"context"
	"fmt"
	di "github.com/cratis/fundamentals.go/dependencyinjection"
	"github.com/cratis/fundamentals.go/dependencyinjection/container"
)

func ExampleResolve() {
	var registry container.Registry
	if err := di.BindValue(&registry, "configured value"); err != nil {
		panic(err)
	}
	p, err := registry.Build()
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	s, err := p.NewScope(ctx)
	if err != nil {
		panic(err)
	}
	value, err := di.Resolve[string](ctx, s)
	if err != nil {
		panic(err)
	}
	fmt.Println(value)
	if err := s.Close(ctx); err != nil {
		panic(err)
	}
	if err := p.Close(ctx); err != nil {
		panic(err)
	}
	// Output: configured value
}
