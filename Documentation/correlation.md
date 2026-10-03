---
title: Carry a correlation ID through an operation
description: Attach, propagate and deliberately clear an operation ID while preserving context cancellation, deadlines and unrelated values.
---

<!-- Copyright (c) Cratis. All rights reserved. -->
<!-- Licensed under the MIT license. See LICENSE file in the project root for full license information. -->

To connect logs from the same operation, attach its correlation ID to the
operation's existing context and pass the returned context onward. The
`correlation` package, introduced in **v0.2.0**, stores metadata only: it does not parse
headers, generate IDs, start work or configure logging.

## Attach and propagate an ID

Use Go 1.26 or later and install
`github.com/cratis/fundamentals.go@v0.3.0` in your module
(see [release status](releases.md)). Save this complete
program as `main.go`, then run `go run .`. The
[executable propagation example](../documentation_examples_test.go) is its source.

```go
package main

import (
 "context"
 "fmt"

 "github.com/cratis/fundamentals.go/concepts"
 "github.com/cratis/fundamentals.go/correlation"
)

func main() {
 id, err := concepts.ParseUUID("00112233-4455-6677-8899-aabbccddeeff")
 if err != nil {
  panic(err)
 }
 operation, cancel := context.WithCancel(context.Background())
 defer cancel()
 ctx := correlation.WithID(operation, id)
 // Pass ctx to the next operation; the parent remains unchanged.
 fmt.Println("save author:", correlation.FromContext(ctx))
 fmt.Println("parent unset:", correlation.FromContext(operation).IsZero())
 cleared := correlation.WithID(ctx, correlation.ID{})
 fmt.Println("child unset:", correlation.FromContext(cleared).IsZero())
 cancel()
 fmt.Println("canceled:", cleared.Err() == context.Canceled)
}
```

Expected output:

```text
save author: 00112233-4455-6677-8899-aabbccddeeff
parent unset: true
child unset: true
canceled: true
```

In a request handler, derive from the incoming request context rather than
`context.Background()`. `WithID` preserves its deadline, cancellation, error
and unrelated values. The ID is copied into an immutable derived context;
the parent is not modified. Concurrent context reads are safe.

## Read or deliberately clear correlation

| Public API | Behavior |
| --- | --- |
| `type ID = concepts.UUID` | Alias, not a distinct domain type; any shared UUID is assignable |
| `WithID(ctx context.Context, id ID) context.Context` | Return a derived context holding the supplied ID |
| `FromContext(ctx context.Context) ID` | Return the stored ID, or zero when absent |

An explicit zero ID **shadows** an inherited nonzero ID. Reading never falls
back past that zero and never generates a replacement. There is no separate
presence result: absence and explicit zero both read as zero. If your ingress
policy needs that distinction, track presence separately at that boundary.
Generate a fresh ID explicitly with `concepts.NewUUID()` only when your policy
calls for it, and handle errors. Both context functions **panic on nil context**;
pass a real caller context, consistent with Go's context APIs.

## Keep lifetime and authority separate

Pass the derived context to downstream calls; do not store it in a long-lived
client or singleton. The caller still owns cancellation and any work it starts.
Attaching an ID does not extend a deadline, cancel work or wait for goroutines.

Correlation is **not authorization, user identity, tenancy or causation**.
Do not trust a supplied ID as proof of who called. Header names, validation,
generation policy and transport propagation belong to the consuming application
or framework. There is no ambient global accessor and no required DI container.
See the [scalar reference](scalars.md#uuid) for parsing and byte-order rules.
