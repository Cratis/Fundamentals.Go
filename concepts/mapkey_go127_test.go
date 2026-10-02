// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

//go:build go1.27

package concepts_test

// mapKeyDecodeUsesText reports whether encoding/json decodes a map key that
// implements both json.Unmarshaler and encoding.TextUnmarshaler through
// UnmarshalText. Go 1.27 does.
const mapKeyDecodeUsesText = true
