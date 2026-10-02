<!-- Copyright (c) Cratis. All rights reserved. -->
<!-- Licensed under the MIT license. See LICENSE file in the project root for full license information. -->

# Scalar contract fixtures

`scalars.json` is copied unchanged from
[Cratis/Arc.Go at 4511966526a3a646538175c20e9de5282907191e](https://github.com/Cratis/Arc.Go/blob/4511966526a3a646538175c20e9de5282907191e/ContractTests/fixtures/v1/scalars.json).
The [original provenance](https://github.com/Cratis/Arc.Go/blob/4511966526a3a646538175c20e9de5282907191e/ContractTests/fixtures/v1/README.md)
describes source-derived expectations, not captured responses from a running
.NET host. Never regenerate these fixtures from Go output.

## Authority

The C# authority is
[Cratis/Fundamentals at d2accc4a79b6bcf2708213c97093ab5ba6c06381](https://github.com/Cratis/Fundamentals/tree/d2accc4a79b6bcf2708213c97093ab5ba6c06381),
plus .NET System.Text.Json's built-in Guid and TimeSpan converters.
The 11 fixtures use Guid `D`, DateOnly `O`, TimeOnly `O` and TimeSpan `c` forms.
Relevant pinned Fundamentals sources:

- `Source/DotNET/Fundamentals/Json/DateOnlyJsonConverter.cs`
- `Source/DotNET/Fundamentals/Json/TimeOnlyJsonConverter.cs`
- `Source/DotNET/Fundamentals.Specs/Json/for_ConceptAsJsonConverter/GuidConcept.cs`
- `Source/DotNET/Fundamentals/Json/JsonValueExtensions.cs` (`TimeSpan.ToString()`)

`../golden_test.go` requires all 11 cases and compares literal JSON bytes.
The additional asymmetric UUID byte-order test in `../boundaries_test.go` uses
`00112233-4455-6677-8899-aabbccddeeff`; it does not modify the copied fixture.
