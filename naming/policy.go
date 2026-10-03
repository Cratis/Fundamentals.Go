// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package naming

import "strings"

// Policy converts property and read-model names without selecting a serializer's
// default. GetReadModelName takes an explicit dot-separated namespace and model
// name rather than a CLR Type. Callers apply explicit storage-name overrides
// before invoking a policy. No policy pluralizes names.
// Implementations supplied by this package are immutable and concurrency-safe.
type Policy interface {
	GetPropertyName(name string) string
	GetReadModelName(namespace, name string) string
}

// Default preserves names, matching DefaultNamingPolicy with pluralization off.
// Its zero value is ready to use and ignores the read-model namespace.
type Default struct{}

// GetPropertyName returns name unchanged, including empty input.
func (Default) GetPropertyName(name string) string { return name }

// GetReadModelName returns name unchanged, ignoring namespace.
func (Default) GetReadModelName(namespace, name string) string { return name }

// CamelCasePolicy applies CamelCase, matching CamelCaseNamingPolicy with
// pluralization off. Its zero value is ready to use. The Policy suffix avoids
// colliding with the CamelCase function.
type CamelCasePolicy struct{}

// GetPropertyName applies the acronym-friendly CamelCase rule to name.
func (CamelCasePolicy) GetPropertyName(name string) string { return CamelCase(name) }

// GetReadModelName applies CamelCase to name, ignoring namespace.
func (CamelCasePolicy) GetReadModelName(namespace, name string) string { return CamelCase(name) }

// Namespaced prefixes read-model names with camel-cased namespace segments.
// Its zero value skips no segments, uses '-' between them, has no prefix, and
// preserves property and model names. The final separator before the model is
// always '-', even with an empty namespace or a different segment separator.
// Values are immutable; use NewNamespaced for non-default configuration.
type Namespaced struct {
	segmentsToSkip int
	separator      rune
	prefix         string
	camelCase      bool
	configured     bool
}

// NewNamespaced configures namespace skipping, the separator between namespace
// segments, a verbatim prefix, and optional CamelCase for properties and models.
// Negative segmentsToSkip behaves like zero, as in LINQ Skip. Namespace segments
// always use CamelCase. The model name is never pluralized. Separator is a Go
// rune (including NUL); invalid runes become U+FFFD when converted to a string.
func NewNamespaced(segmentsToSkip int, separator rune, prefix string, camelCase bool) Namespaced {
	return Namespaced{
		segmentsToSkip: max(segmentsToSkip, 0),
		separator:      separator,
		prefix:         prefix,
		camelCase:      camelCase,
		configured:     true,
	}
}

// GetPropertyName preserves name unless camel casing was requested.
func (p Namespaced) GetPropertyName(name string) string {
	if p.camelCase {
		return CamelCase(name)
	}
	return name
}

// GetReadModelName joins the retained, camel-cased namespace segments, then
// prepends the configured prefix and appends '-' and the optionally cased name.
// Empty namespace segments are retained; an empty namespace yields '-'+name.
func (p Namespaced) GetReadModelName(namespace, name string) string {
	segments := strings.Split(namespace, ".")
	segments = segments[min(p.segmentsToSkip, len(segments)):]
	for i := range segments {
		segments[i] = CamelCase(segments[i])
	}
	separator := '-'
	if p.configured {
		separator = p.separator
	}
	if p.camelCase {
		name = CamelCase(name)
	}
	return p.prefix + strings.Join(segments, string(separator)) + "-" + name
}
