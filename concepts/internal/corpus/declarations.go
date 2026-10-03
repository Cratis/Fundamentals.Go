// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

// Package corpus holds declarations shared by reflect and go/types recognition tests.
package corpus

import (
	"github.com/cratis/fundamentals.go/concepts"
	"reflect"
	"time"
)

// declarationConcept's methods panic to prove discovery is metadata-only, even
// for private types and generic instantiations with unsupported representations.
type declarationConcept[T any] struct{ value T }

func (c declarationConcept[T]) ConceptValue() T            { panic(c.value) }
func (declarationConcept[T]) MarshalText() ([]byte, error) { panic("discovery executed text encoder") }
func (declarationConcept[T]) MarshalJSON() ([]byte, error) { panic("discovery executed JSON encoder") }
func (*declarationConcept[T]) UnmarshalText([]byte) error  { panic("discovery executed text decoder") }
func (*declarationConcept[T]) UnmarshalJSON([]byte) error  { panic("discovery executed JSON decoder") }

type declarationAlias[T any] = declarationConcept[T]
type sharedAlias = concepts.UUID
type dateAlias = concepts.DateOnly
type timeAlias = concepts.TimeOnly
type spanAlias = concepts.TimeSpan
type conceptAlias = declarationConcept[concepts.UUID]
type definedUUID concepts.UUID
type definedDate concepts.DateOnly
type definedTime concepts.TimeOnly
type definedSpan concepts.TimeSpan

// Unmarked calendar types with either value encoder are codec-only, not concepts.
type jsonDate concepts.DateOnly

func (d jsonDate) MarshalJSON() ([]byte, error) { return concepts.DateOnly(d).MarshalJSON() }

type textDate concepts.DateOnly

func (d textDate) MarshalText() ([]byte, error) { return concepts.DateOnly(d).MarshalText() }

type jsonTime concepts.TimeOnly

func (v jsonTime) MarshalJSON() ([]byte, error) { return concepts.TimeOnly(v).MarshalJSON() }

type textTime concepts.TimeOnly

func (v textTime) MarshalText() ([]byte, error) { return concepts.TimeOnly(v).MarshalText() }

type pointerDate concepts.DateOnly

func (d *pointerDate) MarshalJSON() ([]byte, error) { return concepts.DateOnly(*d).MarshalJSON() }

type pointerTime concepts.TimeOnly

func (v *pointerTime) MarshalText() ([]byte, error) { return concepts.TimeOnly(*v).MarshalText() }

type plainName string
type lostMethods declarationConcept[string]
type cyclicPointer *cyclicPointer
type selfConcept struct{}

func (selfConcept) ConceptValue() selfConcept { panic("marker") }

type pointerSelf struct{}

func (pointerSelf) ConceptValue() *pointerSelf { panic("marker") }

type mutualA struct{}
type mutualB struct{}

func (mutualA) ConceptValue() mutualB { panic("marker") }
func (mutualB) ConceptValue() mutualA { panic("marker") }

type pointerMarker struct{}

func (*pointerMarker) ConceptValue() string { panic("marker") }

type noResult struct{}

func (noResult) ConceptValue() { panic("marker") }

type multiResult struct{}

func (multiResult) ConceptValue() (string, error) { panic("marker") }

type argumentMarker struct{}

func (argumentMarker) ConceptValue(string) string { panic("marker") }

type variadicMarker struct{}

func (variadicMarker) ConceptValue(...string) string { panic("marker") }

type markerOnly string

func (markerOnly) ConceptValue() string { panic("marker") }

type wrongText string

func (wrongText) ConceptValue() string { panic("marker") }
func (wrongText) MarshalText() string  { panic("codec") }

type pointerEncoder string

func (pointerEncoder) ConceptValue() string          { panic("marker") }
func (*pointerEncoder) MarshalText() ([]byte, error) { panic("codec") }

type missingJSON string

func (missingJSON) ConceptValue() string         { panic("marker") }
func (missingJSON) MarshalText() ([]byte, error) { panic("codec") }

// Wrongly typed callable JSON fields exercise lookalike codec rejection without
// declaring malformed standard methods, which go vet rejects independently.
type wrongJSON struct{ MarshalJSON func() []byte }

func (wrongJSON) ConceptValue() string         { panic("marker") }
func (wrongJSON) MarshalText() ([]byte, error) { panic("codec") }

type pointerJSONEncoder string

func (pointerJSONEncoder) ConceptValue() string          { panic("marker") }
func (pointerJSONEncoder) MarshalText() ([]byte, error)  { panic("codec") }
func (*pointerJSONEncoder) MarshalJSON() ([]byte, error) { panic("codec") }

type missingTextDecoder string

func (missingTextDecoder) ConceptValue() string         { panic("marker") }
func (missingTextDecoder) MarshalText() ([]byte, error) { panic("codec") }
func (missingTextDecoder) MarshalJSON() ([]byte, error) { panic("codec") }

type wrongTextDecoder string

func (wrongTextDecoder) ConceptValue() string         { panic("marker") }
func (wrongTextDecoder) MarshalText() ([]byte, error) { panic("codec") }
func (wrongTextDecoder) MarshalJSON() ([]byte, error) { panic("codec") }
func (*wrongTextDecoder) UnmarshalText(string) error  { panic("codec") }

type missingJSONDecoder string

func (missingJSONDecoder) ConceptValue() string         { panic("marker") }
func (missingJSONDecoder) MarshalText() ([]byte, error) { panic("codec") }
func (missingJSONDecoder) MarshalJSON() ([]byte, error) { panic("codec") }
func (*missingJSONDecoder) UnmarshalText([]byte) error  { panic("codec") }

type wrongJSONDecoder struct{ UnmarshalJSON func([]byte) }

func (wrongJSONDecoder) ConceptValue() string         { panic("marker") }
func (wrongJSONDecoder) MarshalText() ([]byte, error) { panic("codec") }
func (wrongJSONDecoder) MarshalJSON() ([]byte, error) { panic("codec") }
func (*wrongJSONDecoder) UnmarshalText([]byte) error  { panic("codec") }

type valueTextDecoder string

func (valueTextDecoder) ConceptValue() string         { panic("marker") }
func (valueTextDecoder) MarshalText() ([]byte, error) { panic("codec") }
func (valueTextDecoder) MarshalJSON() ([]byte, error) { panic("codec") }
func (valueTextDecoder) UnmarshalText([]byte) error   { panic("codec") }
func (*valueTextDecoder) UnmarshalJSON([]byte) error  { panic("codec") }

type valueJSONDecoder string

func (valueJSONDecoder) ConceptValue() string         { panic("marker") }
func (valueJSONDecoder) MarshalText() ([]byte, error) { panic("codec") }
func (valueJSONDecoder) MarshalJSON() ([]byte, error) { panic("codec") }
func (*valueJSONDecoder) UnmarshalText([]byte) error  { panic("codec") }
func (valueJSONDecoder) UnmarshalJSON([]byte) error   { panic("codec") }

type codecOnly string

func (codecOnly) MarshalJSON() ([]byte, error) { panic("codec") }

type embeddedConcept struct{ declarationConcept[string] }
type embeddedPointer struct{ *declarationConcept[string] }
type Book struct {
	AuthorID
	EditorID
}
type EditorID AuthorID

var _ concepts.Concept[concepts.UUID] = EditorID{}

func (id EditorID) ConceptValue() concepts.UUID  { return concepts.UUID(id) }
func (id EditorID) MarshalText() ([]byte, error) { return concepts.UUID(id).MarshalText() }
func (id EditorID) MarshalJSON() ([]byte, error) { return concepts.UUID(id).MarshalJSON() }
func (id *EditorID) UnmarshalText(data []byte) error {
	var value concepts.UUID
	if err := value.UnmarshalText(data); err != nil {
		return err
	}
	*id = EditorID(value)
	return nil
}
func (id *EditorID) UnmarshalJSON(data []byte) error {
	var value concepts.UUID
	if err := value.UnmarshalJSON(data); err != nil {
		return err
	}
	*id = EditorID(value)
	return nil
}

type shadowedMarker struct {
	declarationConcept[string]
	ConceptValue string
}
type overriddenMarker struct{ declarationConcept[string] }

func (overriddenMarker) ConceptValue() string { panic("marker") }

type recursiveEmbedding struct{ *recursiveEmbedding }
type markedRecursiveEmbedding struct{ *markedRecursiveEmbedding }

func (markedRecursiveEmbedding) ConceptValue() string { panic("marker") }

type ordinaryModel struct{ Value declarationConcept[string] }

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

// DeclarationConcept keeps existing runtime test helpers on the same declarations.
type DeclarationConcept[T any] = declarationConcept[T]

// DefinedUUID is an unmarked UUID-derived declaration.
type DefinedUUID = definedUUID

// PlainName is an ordinary named primitive.
type PlainName = plainName

// CyclicPointer is a named pointer cycle.
type CyclicPointer = cyclicPointer

// DefinedDate is an unmarked date-derived declaration.
type DefinedDate = definedDate

// DefinedTime is an unmarked time-derived declaration.
type DefinedTime = definedTime

// DefinedSpan is an unmarked duration-derived declaration.
type DefinedSpan = definedSpan

type case001 = sharedAlias
type scalar001 = concepts.UUID
type case002 = dateAlias
type scalar002 = concepts.DateOnly
type case003 = timeAlias
type scalar003 = concepts.TimeOnly
type case004 = spanAlias
type scalar004 = concepts.TimeSpan
type case005 = declarationConcept[byte]
type scalar005 = uint8
type case006 = declarationConcept[rune]
type scalar006 = int32
type case007 = conceptAlias
type scalar007 = concepts.UUID
type case008 = declarationAlias[string]
type scalar008 = string
type case009 = definedUUID
type case010 = definedDate
type case011 = definedTime
type case012 = jsonDate
type case013 = textDate
type case014 = jsonTime
type case015 = textTime
type case016 = pointerDate
type case017 = pointerTime
type case018 = definedSpan
type case019 = string
type case020 = plainName
type case021 = lostMethods
type case022 = cyclicPointer
type case023 = selfConcept
type scalar023 = selfConcept
type case024 = pointerSelf
type scalar024 = *pointerSelf
type case025 = mutualA
type scalar025 = mutualB
type case026 = mutualB
type scalar026 = mutualA
type case027 = declarationConcept[conceptAlias]
type scalar027 = conceptAlias
type case028 = declarationConcept[*conceptAlias]
type scalar028 = *conceptAlias
type case029 = declarationConcept[cyclicPointer]
type scalar029 = cyclicPointer
type case030 = pointerMarker
type case031 = concepts.Concept[string]
type case032 = noResult
type case033 = multiResult
type case034 = argumentMarker
type case035 = variadicMarker
type case036 = markerOnly
type scalar036 = string
type case037 = wrongText
type scalar037 = string
type case038 = pointerEncoder
type scalar038 = string
type case039 = missingJSON
type scalar039 = string
type case040 = wrongJSON
type scalar040 = string
type case041 = pointerJSONEncoder
type scalar041 = string
type case042 = missingTextDecoder
type scalar042 = string
type case043 = wrongTextDecoder
type scalar043 = string
type case044 = missingJSONDecoder
type scalar044 = string
type case045 = wrongJSONDecoder
type scalar045 = string
type case046 = valueTextDecoder
type scalar046 = string
type case047 = valueJSONDecoder
type scalar047 = string
type case048 = codecOnly
type case049 = embeddedConcept
type case050 = embeddedPointer
type case051 = Book
type case052 = shadowedMarker
type case053 = overriddenMarker
type case054 = recursiveEmbedding
type case055 = markedRecursiveEmbedding
type case056 = ordinaryModel
type case057 = []conceptAlias
type case058 = map[string]conceptAlias
type case059 = any
type case060 = declarationConcept[string]
type scalar060 = string
type case061 = declarationConcept[bool]
type scalar061 = bool
type case062 = declarationConcept[int]
type scalar062 = int
type case063 = declarationConcept[int8]
type scalar063 = int8
type case064 = declarationConcept[int16]
type scalar064 = int16
type case065 = declarationConcept[int32]
type scalar065 = int32
type case066 = declarationConcept[int64]
type scalar066 = int64
type case067 = declarationConcept[uint]
type scalar067 = uint
type case068 = declarationConcept[uint8]
type scalar068 = uint8
type case069 = declarationConcept[uint16]
type scalar069 = uint16
type case070 = declarationConcept[uint32]
type scalar070 = uint32
type case071 = declarationConcept[uint64]
type scalar071 = uint64
type case072 = declarationConcept[float32]
type scalar072 = float32
type case073 = declarationConcept[float64]
type scalar073 = float64
type case074 = declarationConcept[concepts.UUID]
type scalar074 = concepts.UUID
type case075 = declarationConcept[concepts.DateOnly]
type scalar075 = concepts.DateOnly
type case076 = declarationConcept[concepts.TimeOnly]
type scalar076 = concepts.TimeOnly
type case077 = declarationConcept[concepts.TimeSpan]
type scalar077 = concepts.TimeSpan
type case078 = concepts.UUID
type scalar078 = concepts.UUID
type case079 = concepts.DateOnly
type scalar079 = concepts.DateOnly
type case080 = concepts.TimeOnly
type scalar080 = concepts.TimeOnly
type case081 = concepts.TimeSpan
type scalar081 = concepts.TimeSpan
type case082 = declarationConcept[plainName]
type scalar082 = plainName
type case083 = declarationConcept[uintptr]
type scalar083 = uintptr
type case084 = declarationConcept[complex64]
type scalar084 = complex64
type case085 = declarationConcept[complex128]
type scalar085 = complex128
type case086 = declarationConcept[*string]
type scalar086 = *string
type case087 = declarationConcept[*concepts.UUID]
type scalar087 = *concepts.UUID
type case088 = declarationConcept[[16]byte]
type scalar088 = [16]byte
type case089 = declarationConcept[struct{}]
type scalar089 = struct{}
type case090 = declarationConcept[[]string]
type scalar090 = []string
type case091 = declarationConcept[map[string]string]
type scalar091 = map[string]string
type case092 = declarationConcept[any]
type scalar092 = any
type case093 = declarationConcept[interface{ Read([]byte) (int, error) }]
type scalar093 = interface{ Read([]byte) (int, error) }
type case094 = declarationConcept[concepts.Concept[string]]
type scalar094 = concepts.Concept[string]
type case095 = declarationConcept[time.Time]
type scalar095 = time.Time
type case096 = declarationConcept[definedUUID]
type scalar096 = definedUUID
type case097 = *conceptAlias
type scalar097 = concepts.UUID
type case098 = **conceptAlias
type scalar098 = concepts.UUID

// Case describes one declaration and its independently specified expected outcome.
// TypeName and ScalarName refer to aliases in this package's go/types scope.
// An empty TypeName represents nil input; an empty ScalarName represents nil metadata.
// KindInvalid with no Reason denotes not-a-concept.
type Case struct {
	Name, TypeName, ScalarName string
	Input, Scalar              reflect.Type
	Kind                       concepts.ScalarKind
	Reason                     concepts.InvalidReason
	Method                     string
	PointerDepth               int
}

// Cases returns a fresh table shared by the runtime and compile-time tests.
func Cases() []Case {
	return append([]Case{
		{"nil", "", "", nil, nil, concepts.KindInvalid, concepts.ReasonNilType, "", 0},
		{"shared alias", "case001", "scalar001", reflect.TypeFor[case001](), reflect.TypeFor[scalar001](), concepts.KindUUID, "", "", 0},
		{"date alias", "case002", "scalar002", reflect.TypeFor[case002](), reflect.TypeFor[scalar002](), concepts.KindDateOnly, "", "", 0},
		{"time alias", "case003", "scalar003", reflect.TypeFor[case003](), reflect.TypeFor[scalar003](), concepts.KindTimeOnly, "", "", 0},
		{"span alias", "case004", "scalar004", reflect.TypeFor[case004](), reflect.TypeFor[scalar004](), concepts.KindTimeSpan, "", "", 0},
		{"byte alias result", "case005", "scalar005", reflect.TypeFor[case005](), reflect.TypeFor[scalar005](), concepts.KindUint8, "", "", 0},
		{"rune alias result", "case006", "scalar006", reflect.TypeFor[case006](), reflect.TypeFor[scalar006](), concepts.KindInt32, "", "", 0},
		{"concept alias", "case007", "scalar007", reflect.TypeFor[case007](), reflect.TypeFor[scalar007](), concepts.KindUUID, "", "", 0},
		{"generic alias", "case008", "scalar008", reflect.TypeFor[case008](), reflect.TypeFor[scalar008](), concepts.KindString, "", "", 0},
		{"UUID without methods", "case009", "", reflect.TypeFor[case009](), nil, concepts.KindInvalid, "", "", 0},
		{"date without forwarding", "case010", "", reflect.TypeFor[case010](), nil, concepts.KindInvalid, concepts.ReasonMissingForwarding, "ConceptValue", 0},
		{"time without forwarding", "case011", "", reflect.TypeFor[case011](), nil, concepts.KindInvalid, concepts.ReasonMissingForwarding, "ConceptValue", 0},
		{"date JSON codec only", "case012", "", reflect.TypeFor[case012](), nil, concepts.KindInvalid, "", "", 0},
		{"date text codec only", "case013", "", reflect.TypeFor[case013](), nil, concepts.KindInvalid, "", "", 0},
		{"time JSON codec only", "case014", "", reflect.TypeFor[case014](), nil, concepts.KindInvalid, "", "", 0},
		{"time text codec only", "case015", "", reflect.TypeFor[case015](), nil, concepts.KindInvalid, "", "", 0},
		{"date pointer encoder only", "case016", "", reflect.TypeFor[case016](), nil, concepts.KindInvalid, concepts.ReasonMissingForwarding, "ConceptValue", 0},
		{"time pointer encoder only", "case017", "", reflect.TypeFor[case017](), nil, concepts.KindInvalid, concepts.ReasonMissingForwarding, "ConceptValue", 0},
		{"duration limitation", "case018", "", reflect.TypeFor[case018](), nil, concepts.KindInvalid, "", "", 0},
		{"plain primitive", "case019", "", reflect.TypeFor[case019](), nil, concepts.KindInvalid, "", "", 0},
		{"plain named primitive", "case020", "", reflect.TypeFor[case020](), nil, concepts.KindInvalid, "", "", 0},
		{"lost methods", "case021", "", reflect.TypeFor[case021](), nil, concepts.KindInvalid, "", "", 0},
		{"cyclic pointer", "case022", "", reflect.TypeFor[case022](), nil, concepts.KindInvalid, concepts.ReasonRecursiveType, "", 0},
		{"self", "case023", "scalar023", reflect.TypeFor[case023](), reflect.TypeFor[scalar023](), concepts.KindInvalid, concepts.ReasonRecursiveType, "ConceptValue", 0},
		{"pointer self", "case024", "scalar024", reflect.TypeFor[case024](), reflect.TypeFor[scalar024](), concepts.KindInvalid, concepts.ReasonRecursiveType, "ConceptValue", 0},
		{"mutual A", "case025", "scalar025", reflect.TypeFor[case025](), reflect.TypeFor[scalar025](), concepts.KindInvalid, concepts.ReasonNestedConcept, "ConceptValue", 0},
		{"mutual B", "case026", "scalar026", reflect.TypeFor[case026](), reflect.TypeFor[scalar026](), concepts.KindInvalid, concepts.ReasonNestedConcept, "ConceptValue", 0},
		{"nested", "case027", "scalar027", reflect.TypeFor[case027](), reflect.TypeFor[scalar027](), concepts.KindInvalid, concepts.ReasonNestedConcept, "ConceptValue", 0},
		{"pointer nested", "case028", "scalar028", reflect.TypeFor[case028](), reflect.TypeFor[scalar028](), concepts.KindInvalid, concepts.ReasonNestedConcept, "ConceptValue", 0},
		{"recursive result", "case029", "scalar029", reflect.TypeFor[case029](), reflect.TypeFor[scalar029](), concepts.KindInvalid, concepts.ReasonRecursiveType, "ConceptValue", 0},
		{"pointer marker", "case030", "", reflect.TypeFor[case030](), nil, concepts.KindInvalid, concepts.ReasonPointerMethod, "ConceptValue", 0},
		{"interface marker", "case031", "", reflect.TypeFor[case031](), nil, concepts.KindInvalid, concepts.ReasonInterface, "ConceptValue", 0},
		{"no result", "case032", "", reflect.TypeFor[case032](), nil, concepts.KindInvalid, concepts.ReasonInvalidMethod, "ConceptValue", 0},
		{"multiple results", "case033", "", reflect.TypeFor[case033](), nil, concepts.KindInvalid, concepts.ReasonInvalidMethod, "ConceptValue", 0},
		{"parameter", "case034", "", reflect.TypeFor[case034](), nil, concepts.KindInvalid, concepts.ReasonInvalidMethod, "ConceptValue", 0},
		{"variadic", "case035", "", reflect.TypeFor[case035](), nil, concepts.KindInvalid, concepts.ReasonInvalidMethod, "ConceptValue", 0},
		{"marker only", "case036", "scalar036", reflect.TypeFor[case036](), reflect.TypeFor[scalar036](), concepts.KindInvalid, concepts.ReasonMissingCodec, "MarshalText", 0},
		{"wrong text encoder", "case037", "scalar037", reflect.TypeFor[case037](), reflect.TypeFor[scalar037](), concepts.KindInvalid, concepts.ReasonMissingCodec, "MarshalText", 0},
		{"pointer encoder", "case038", "scalar038", reflect.TypeFor[case038](), reflect.TypeFor[scalar038](), concepts.KindInvalid, concepts.ReasonMissingCodec, "MarshalText", 0},
		{"missing JSON encoder", "case039", "scalar039", reflect.TypeFor[case039](), reflect.TypeFor[scalar039](), concepts.KindInvalid, concepts.ReasonMissingCodec, "MarshalJSON", 0},
		{"wrong JSON encoder field", "case040", "scalar040", reflect.TypeFor[case040](), reflect.TypeFor[scalar040](), concepts.KindInvalid, concepts.ReasonMissingCodec, "MarshalJSON", 0},
		{"pointer JSON encoder", "case041", "scalar041", reflect.TypeFor[case041](), reflect.TypeFor[scalar041](), concepts.KindInvalid, concepts.ReasonMissingCodec, "MarshalJSON", 0},
		{"missing text decoder", "case042", "scalar042", reflect.TypeFor[case042](), reflect.TypeFor[scalar042](), concepts.KindInvalid, concepts.ReasonMissingCodec, "UnmarshalText", 0},
		{"wrong text decoder", "case043", "scalar043", reflect.TypeFor[case043](), reflect.TypeFor[scalar043](), concepts.KindInvalid, concepts.ReasonMissingCodec, "UnmarshalText", 0},
		{"missing JSON decoder", "case044", "scalar044", reflect.TypeFor[case044](), reflect.TypeFor[scalar044](), concepts.KindInvalid, concepts.ReasonMissingCodec, "UnmarshalJSON", 0},
		{"wrong JSON decoder field", "case045", "scalar045", reflect.TypeFor[case045](), reflect.TypeFor[scalar045](), concepts.KindInvalid, concepts.ReasonMissingCodec, "UnmarshalJSON", 0},
		{"value text decoder", "case046", "scalar046", reflect.TypeFor[case046](), reflect.TypeFor[scalar046](), concepts.KindInvalid, concepts.ReasonValueUnmarshaler, "UnmarshalText", 0},
		{"value JSON decoder", "case047", "scalar047", reflect.TypeFor[case047](), reflect.TypeFor[scalar047](), concepts.KindInvalid, concepts.ReasonValueUnmarshaler, "UnmarshalJSON", 0},
		{"codec only", "case048", "", reflect.TypeFor[case048](), nil, concepts.KindInvalid, "", "", 0},
		{"value embedding", "case049", "", reflect.TypeFor[case049](), nil, concepts.KindInvalid, concepts.ReasonEmbeddedFields, "", 0},
		{"pointer embedding", "case050", "", reflect.TypeFor[case050](), nil, concepts.KindInvalid, concepts.ReasonEmbeddedFields, "", 0},
		{"ambiguous Book", "case051", "", reflect.TypeFor[case051](), nil, concepts.KindInvalid, "", "", 0},
		{"shadowed marker", "case052", "", reflect.TypeFor[case052](), nil, concepts.KindInvalid, "", "", 0},
		{"override with embedding", "case053", "", reflect.TypeFor[case053](), nil, concepts.KindInvalid, concepts.ReasonEmbeddedFields, "", 0},
		{"recursive embedding", "case054", "", reflect.TypeFor[case054](), nil, concepts.KindInvalid, "", "", 0},
		{"marked recursive embedding", "case055", "", reflect.TypeFor[case055](), nil, concepts.KindInvalid, concepts.ReasonEmbeddedFields, "", 0},
		{"ordinary fields", "case056", "", reflect.TypeFor[case056](), nil, concepts.KindInvalid, "", "", 0},
		{"ordinary slice", "case057", "", reflect.TypeFor[case057](), nil, concepts.KindInvalid, "", "", 0},
		{"ordinary map", "case058", "", reflect.TypeFor[case058](), nil, concepts.KindInvalid, "", "", 0},
		{"ordinary interface", "case059", "", reflect.TypeFor[case059](), nil, concepts.KindInvalid, "", "", 0},
		{"allowlist string", "case060", "scalar060", reflect.TypeFor[case060](), reflect.TypeFor[scalar060](), concepts.KindString, "", "", 0},
		{"allowlist bool", "case061", "scalar061", reflect.TypeFor[case061](), reflect.TypeFor[scalar061](), concepts.KindBool, "", "", 0},
		{"allowlist int", "case062", "scalar062", reflect.TypeFor[case062](), reflect.TypeFor[scalar062](), concepts.KindInt, "", "", 0},
		{"allowlist int8", "case063", "scalar063", reflect.TypeFor[case063](), reflect.TypeFor[scalar063](), concepts.KindInt8, "", "", 0},
		{"allowlist int16", "case064", "scalar064", reflect.TypeFor[case064](), reflect.TypeFor[scalar064](), concepts.KindInt16, "", "", 0},
		{"allowlist int32", "case065", "scalar065", reflect.TypeFor[case065](), reflect.TypeFor[scalar065](), concepts.KindInt32, "", "", 0},
		{"allowlist int64", "case066", "scalar066", reflect.TypeFor[case066](), reflect.TypeFor[scalar066](), concepts.KindInt64, "", "", 0},
		{"allowlist uint", "case067", "scalar067", reflect.TypeFor[case067](), reflect.TypeFor[scalar067](), concepts.KindUint, "", "", 0},
		{"allowlist uint8", "case068", "scalar068", reflect.TypeFor[case068](), reflect.TypeFor[scalar068](), concepts.KindUint8, "", "", 0},
		{"allowlist uint16", "case069", "scalar069", reflect.TypeFor[case069](), reflect.TypeFor[scalar069](), concepts.KindUint16, "", "", 0},
		{"allowlist uint32", "case070", "scalar070", reflect.TypeFor[case070](), reflect.TypeFor[scalar070](), concepts.KindUint32, "", "", 0},
		{"allowlist uint64", "case071", "scalar071", reflect.TypeFor[case071](), reflect.TypeFor[scalar071](), concepts.KindUint64, "", "", 0},
		{"allowlist float32", "case072", "scalar072", reflect.TypeFor[case072](), reflect.TypeFor[scalar072](), concepts.KindFloat32, "", "", 0},
		{"allowlist float64", "case073", "scalar073", reflect.TypeFor[case073](), reflect.TypeFor[scalar073](), concepts.KindFloat64, "", "", 0},
		{"allowlist concepts.UUID", "case074", "scalar074", reflect.TypeFor[case074](), reflect.TypeFor[scalar074](), concepts.KindUUID, "", "", 0},
		{"allowlist concepts.DateOnly", "case075", "scalar075", reflect.TypeFor[case075](), reflect.TypeFor[scalar075](), concepts.KindDateOnly, "", "", 0},
		{"allowlist concepts.TimeOnly", "case076", "scalar076", reflect.TypeFor[case076](), reflect.TypeFor[scalar076](), concepts.KindTimeOnly, "", "", 0},
		{"allowlist concepts.TimeSpan", "case077", "scalar077", reflect.TypeFor[case077](), reflect.TypeFor[scalar077](), concepts.KindTimeSpan, "", "", 0},
		{"shared concepts.UUID", "case078", "scalar078", reflect.TypeFor[case078](), reflect.TypeFor[scalar078](), concepts.KindUUID, "", "", 0},
		{"shared concepts.DateOnly", "case079", "scalar079", reflect.TypeFor[case079](), reflect.TypeFor[scalar079](), concepts.KindDateOnly, "", "", 0},
		{"shared concepts.TimeOnly", "case080", "scalar080", reflect.TypeFor[case080](), reflect.TypeFor[scalar080](), concepts.KindTimeOnly, "", "", 0},
		{"shared concepts.TimeSpan", "case081", "scalar081", reflect.TypeFor[case081](), reflect.TypeFor[scalar081](), concepts.KindTimeSpan, "", "", 0},
		{"unsupported plainName", "case082", "scalar082", reflect.TypeFor[case082](), reflect.TypeFor[scalar082](), concepts.KindInvalid, concepts.ReasonUnsupportedType, "ConceptValue", 0},
		{"unsupported uintptr", "case083", "scalar083", reflect.TypeFor[case083](), reflect.TypeFor[scalar083](), concepts.KindInvalid, concepts.ReasonUnsupportedType, "ConceptValue", 0},
		{"unsupported complex64", "case084", "scalar084", reflect.TypeFor[case084](), reflect.TypeFor[scalar084](), concepts.KindInvalid, concepts.ReasonUnsupportedType, "ConceptValue", 0},
		{"unsupported complex128", "case085", "scalar085", reflect.TypeFor[case085](), reflect.TypeFor[scalar085](), concepts.KindInvalid, concepts.ReasonUnsupportedType, "ConceptValue", 0},
		{"unsupported *string", "case086", "scalar086", reflect.TypeFor[case086](), reflect.TypeFor[scalar086](), concepts.KindInvalid, concepts.ReasonUnsupportedType, "ConceptValue", 0},
		{"unsupported *concepts.UUID", "case087", "scalar087", reflect.TypeFor[case087](), reflect.TypeFor[scalar087](), concepts.KindInvalid, concepts.ReasonUnsupportedType, "ConceptValue", 0},
		{"unsupported [16]byte", "case088", "scalar088", reflect.TypeFor[case088](), reflect.TypeFor[scalar088](), concepts.KindInvalid, concepts.ReasonUnsupportedType, "ConceptValue", 0},
		{"unsupported struct{}", "case089", "scalar089", reflect.TypeFor[case089](), reflect.TypeFor[scalar089](), concepts.KindInvalid, concepts.ReasonUnsupportedType, "ConceptValue", 0},
		{"unsupported []string", "case090", "scalar090", reflect.TypeFor[case090](), reflect.TypeFor[scalar090](), concepts.KindInvalid, concepts.ReasonUnsupportedType, "ConceptValue", 0},
		{"unsupported map[string]string", "case091", "scalar091", reflect.TypeFor[case091](), reflect.TypeFor[scalar091](), concepts.KindInvalid, concepts.ReasonUnsupportedType, "ConceptValue", 0},
		{"unsupported any", "case092", "scalar092", reflect.TypeFor[case092](), reflect.TypeFor[scalar092](), concepts.KindInvalid, concepts.ReasonUnsupportedType, "ConceptValue", 0},
		{"unsupported interface{ Read([]byte) (int, error) }", "case093", "scalar093", reflect.TypeFor[case093](), reflect.TypeFor[scalar093](), concepts.KindInvalid, concepts.ReasonUnsupportedType, "ConceptValue", 0},
		{"unsupported concepts.Concept[string]", "case094", "scalar094", reflect.TypeFor[case094](), reflect.TypeFor[scalar094](), concepts.KindInvalid, concepts.ReasonUnsupportedType, "ConceptValue", 0},
		{"unsupported time.Time", "case095", "scalar095", reflect.TypeFor[case095](), reflect.TypeFor[scalar095](), concepts.KindInvalid, concepts.ReasonUnsupportedType, "ConceptValue", 0},
		{"unsupported definedUUID", "case096", "scalar096", reflect.TypeFor[case096](), reflect.TypeFor[scalar096](), concepts.KindInvalid, concepts.ReasonUnsupportedType, "ConceptValue", 0},
		{"pointer concept", "case097", "scalar097", reflect.TypeFor[case097](), reflect.TypeFor[scalar097](), concepts.KindUUID, "", "", 1},
		{"double pointer concept", "case098", "scalar098", reflect.TypeFor[case098](), reflect.TypeFor[scalar098](), concepts.KindUUID, "", "", 2},
	}, codecCases()...)
}
