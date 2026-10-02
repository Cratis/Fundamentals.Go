// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

//go:build goexperiment.jsonv2

package concepts_test

// mapKeyDecodeUsesText reports whether encoding/json decodes a map key that
// implements both json.Unmarshaler and encoding.TextUnmarshaler through
// UnmarshalText. The jsonv2-backed implementation (the Go 1.27 default) does.
const mapKeyDecodeUsesText = true
