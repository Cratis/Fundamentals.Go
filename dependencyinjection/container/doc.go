// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

// Package container implements the optional default dependency injection provider.
// It is extracted from Arc.Go services at
// 304e97466bca594b51a3d7bbc723f4c477a6a99c, with framework metadata replaced by
// generic context guards. Registry is single-owner; built providers and ordinary
// scopes support concurrent use. There is no root resolver or ambient provider.
//
// Owned values are closed in reverse creation order. Cleanup prefers the
// unexported interface { Close(context.Context) error } over io.Closer. Closing
// releases resources, never commits application effects. Stop and join application
// work first: Close joins admitted resolutions, not handlers. Cancellation before
// cleanup can be resumed; once cleanup starts, all callbacks run synchronously and
// cooperatively. Results are retained and cleanup does not repeat. Failed non-nil
// owned results get a separate cooperative 30-second cleanup context.
package container
