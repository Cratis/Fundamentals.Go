// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

// This probe records invocation, then lets go/packages fall back to go list.
package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	if _, err := io.Copy(io.Discard, os.Stdin); err != nil {
		os.Exit(1)
	}
	if err := os.WriteFile(os.Getenv("PACKAGESLOADING_DRIVER_MARKER"), []byte("invoked"), 0o600); err != nil {
		os.Exit(1)
	}
	if _, err := fmt.Println(`{"NotHandled":true}`); err != nil {
		os.Exit(1)
	}
}
