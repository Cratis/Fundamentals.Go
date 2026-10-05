// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package naming_test

import (
	"fmt"

	"github.com/cratis/fundamentals.go/naming"
)

func Example_individualNames() {
	fmt.Println(naming.CamelCase("Person"))
	fmt.Println(naming.CamelCase("URLValue"))
	fmt.Println(naming.CamelCase("ABCdef"))
	fmt.Println(naming.PascalCase("person_name"))
	// Output:
	// person
	// URLValue
	// ABCdef
	// Person_name
}

func Example_namespacedStorage() {
	policy := naming.NewNamespaced(1, '/', "app:", true)
	fmt.Println(policy.GetReadModelName("Cratis.ReadModels.People", "Person"))
	fmt.Println(policy.GetPropertyName("Person"))
	// Output:
	// app:readModels/people-person
	// person
}

func ExampleCamelCase() {
	fmt.Println(naming.CamelCase("Person"))
	fmt.Println(naming.CamelCase("URLValue"))
	// Output:
	// person
	// URLValue
}

func ExamplePascalCase() {
	fmt.Println(naming.PascalCase("person_name"))
	// Output: Person_name
}

func ExamplePolicy() {
	for _, policy := range []naming.Policy{naming.Default{}, naming.CamelCasePolicy{}, naming.Namespaced{}} {
		fmt.Println(policy.GetPropertyName("Person"), policy.GetReadModelName("Cratis.Models", "Person"))
	}
	// Output:
	// Person Person
	// person person
	// Person cratis-models-Person
}

func ExampleNewNamespaced() {
	policy := naming.NewNamespaced(1, '/', "app:", true)
	fmt.Println(policy.GetReadModelName("Cratis.ReadModels.People", "Person"))
	fmt.Println(policy.GetReadModelName("", "Person"))
	// Output:
	// app:readModels/people-person
	// app:-person
}
