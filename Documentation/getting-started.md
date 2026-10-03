---
title: Keep domain identity in a JSON round trip
description: Install Fundamentals.Go v0.3.0 and round-trip a UUID-backed author ID using plain Go and encoding/json.
---

<!-- Copyright (c) Cratis. All rights reserved. -->
<!-- Licensed under the MIT license. See LICENSE file in the project root for full license information. -->

An author's ID should not be interchangeable with every string or UUID in your
program. In this tutorial you define an `AuthorID`, encode an author as JSON,
and decode it without losing either the value or its Go type. No container,
server or C# knowledge is required.

This tutorial targets Fundamentals.Go **v0.3.0, an experimental v0.x release**.
Minor releases may change APIs with migration notes. This example uses only the
standard-library-only root module.

## Create a small Go program

With Go **1.26 or later**, run these commands in a new directory;
see [release status](releases.md):

```sh
mkdir author-example
cd author-example
go mod init example.com/author-example
go get github.com/cratis/fundamentals.go@v0.3.0
```

Use the lowercase import path even though the GitHub repository name has capital
letters. You now have a module pinned to the version this tutorial uses.

## Define the ID and round-trip an author

Save this complete program as `main.go`. It comes from the executable
[domain round-trip example](../documentation_examples_test.go).

```go
package main

import (
 "encoding/json"
 "fmt"

 "github.com/cratis/fundamentals.go/concepts"
)

type AuthorID concepts.UUID

var _ concepts.Concept[concepts.UUID] = AuthorID{}

func (id AuthorID) ConceptValue() concepts.UUID  { return concepts.UUID(id) }
func (id AuthorID) MarshalText() ([]byte, error) { return concepts.UUID(id).MarshalText() }
func (id AuthorID) MarshalJSON() ([]byte, error) { return concepts.UUID(id).MarshalJSON() }
func (id *AuthorID) UnmarshalText(data []byte) error {
 var value concepts.UUID
 if err := value.UnmarshalText(data); err != nil {
  return err
 }
 *id = AuthorID(value)
 return nil
}
func (id *AuthorID) UnmarshalJSON(data []byte) error {
 var value concepts.UUID
 if err := value.UnmarshalJSON(data); err != nil {
  return err
 }
 *id = AuthorID(value)
 return nil
}

func main() {
 id, err := concepts.ParseUUID("00112233-4455-6677-8899-AABBCCDDEEFF")
 if err != nil {
  panic(err)
 }
 type author struct {
  ID   AuthorID `json:"id"`
  Name string   `json:"name"`
 }
 original := author{ID: AuthorID(id), Name: "Ada"}
 data, err := json.Marshal(original)
 if err != nil {
  panic(err)
 }
 var restored author
 if err := json.Unmarshal(data, &restored); err != nil {
  panic(err)
 }
 fmt.Println(string(data))
 fmt.Println("same author:", restored == original)
}
```

`AuthorID` is a defined type, not an alias. It keeps the declared domain identity;
`ConceptValue` names its exact scalar representation, `concepts.UUID` rather
than an arbitrary `[16]byte` or string. The compile-time assertion checks that
marker, not the correctness of your codecs.

Go defined types do **not** inherit methods. Without the forwarding methods,
a UUID-derived value can encode as a byte array instead of a UUID string. Value
receivers encode even unaddressable values, such as map entries. Pointer decoders
parse into a temporary and assign only after success, preserving the old value
on malformed input. JSON field names come from explicit tags, not a global
naming policy. The example panics only to stop a failed demonstration; application
code should return or handle these errors at its boundary.

## Run and inspect the result

```sh
go run .
```

Expected output:

```text
{"id":"00112233-4455-6677-8899-aabbccddeeff","name":"Ada"}
same author: true
```

The uppercase input normalizes to lowercase dashed text. The JSON contains a
scalar string, while the decoded struct still holds an `AuthorID`. The equality
check verifies the complete round trip rather than only successful encoding.
To create an ID instead of reading one, call `concepts.NewUUID()` and handle its
error before converting the result to `AuthorID`.

## Choose the next step

- [Scalar reference](scalars.md): exact UUID/date/time/duration formats, ranges,
  null behavior and UUID SQL support.
- [Typed concepts](concepts.md): validate declarations with `Underlying`, check
  encoded bytes with `CheckJSON`, or recognize types in a generator.
- [Correlation context](correlation.md): carry an operation ID without globals.
- [Dependency injection](dependency-injection.md): optional composition and
  ownership when plain constructors are no longer enough.

There is no automatic validation of business rules or universal serializer.
A syntactically valid UUID may still be the wrong author for your operation;
that check belongs to your application.
