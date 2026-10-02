// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

// Package dependencyinjection defines container-neutral exact-type registration,
// resolution and scope contracts. Plain constructors and custom Resolver or
// ScopeFactory implementations need no container. Import this package as di.
//
// The default container is optional, in dependencyinjection/container. Its
// extraction baseline is Arc.Go services at
// 304e97466bca594b51a3d7bbc723f4c477a6a99c.
package dependencyinjection
