// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.
using System.Globalization;
using System.Reflection;
using System.Runtime.InteropServices;
using System.Text;
using System.Text.Json;
using Cratis.Concepts;
using Cratis.Json;

CultureInfo.CurrentCulture = CultureInfo.InvariantCulture;
CultureInfo.CurrentUICulture = CultureInfo.InvariantCulture;
var options = new JsonSerializerOptions();
options.Converters.Add(new EnumConverterFactory());
options.Converters.Add(new ConceptAsJsonConverterFactory());
var results = new List<object>();
var families = new (Type Enum, Type Concept, string[] Inputs, object[] Writes)[] {
    (typeof(Plain), typeof(PlainConcept), ["0", "1", "9", "-1", "2147483647", "-2147483648", "2147483648", "-2147483649", "\"One\"", "\"one\"", "\"ONE\"", "\" One \"", "\"Missing\"", "\"9\"", "\"+1\"", "\"01\"", "\" 9 \"", "\"2147483648\"", "\"-2147483648\"", "\"One, Two\"", "\"one, two\"", "\"One,One\"", "\"1, 2\"", "\"One,\"", "\"\"", "\" \"", "null", "true", "false", "[]", "{}", "1.0", "1e0", "1E+0", "0.5", "\"1.0\"", "\"1e0\"", "-0", "\"-0\""], [0, 1, 3, 9, -1, int.MinValue, int.MaxValue]),
    (typeof(NoZero), typeof(NoZeroConcept), ["0", "\"0\"", "null"], [0]),
    (typeof(Bits), typeof(BitsConcept), ["0", "3", "5", "7", "8", "\"A, B\"", "\"A, C\"", "\"a, c\"", "\"5\"", "\"8\"", "\"A, Missing\""], [0, 3, 5, 7, 8]),
    (typeof(ByteEnum), typeof(ByteConcept), ["0", "1", "255", "256", "-1", "\"One\"", "\"one\"", "\"255\"", "\"256\"", "\"-1\""], [(byte)0, (byte)1, (byte)255]),
    (typeof(UIntEnum), typeof(UIntConcept), ["0", "1", "2147483647", "2147483648", "4294967295", "-1", "\"High\"", "\"4294967295\"", "\"-1\""], [0u, 1u, 2147483647u, 2147483648u, uint.MaxValue]),
    (typeof(LongEnum), typeof(LongConcept), ["0", "1", "-1", "2147483647", "2147483648", "-2147483649", "\"High\"", "\"2147483648\"", "\"-2147483649\"", "\"9223372036854775807\"", "\"9223372036854775808\""], [0L, 1L, -1L, (long)int.MaxValue, 2147483648L, -2147483649L, long.MaxValue]),
};
foreach (var family in families)
{
    foreach (var input in family.Inputs)
        foreach (var (mode, type) in new[] { ("bare", family.Enum), ("concept", family.Concept) })
        {
            object? value;
            try { value = JsonSerializer.Deserialize(input, type, options); }
            catch (Exception error) { results.Add(new { operation = "read", mode, enumType = family.Enum.Name, input, accepted = false, error = Error(error) }); continue; }
            var scalar = value is null ? null : mode == "bare" ? value : value.GetConceptValue();
            results.Add(new { operation = "read", mode, enumType = family.Enum.Name, input, accepted = true, numeric = scalar is null ? null : Numeric(scalar), display = scalar?.ToString(), output = Serialize(value, type) });
        }
    foreach (var number in family.Writes)
        foreach (var (mode, type) in new[] { ("bare", family.Enum), ("concept", family.Concept) })
        {
            var scalar = Enum.ToObject(family.Enum, number);
            var value = mode == "bare" ? scalar : Activator.CreateInstance(type, scalar);
            results.Add(new { operation = "write", mode, enumType = family.Enum.Name, numeric = Numeric(scalar), output = Serialize(value, type) });
        }
}
try
{
    var value = JsonSerializer.Deserialize<Plain?>("null", options);
    results.Add(new { operation = "read", mode = "nullable-bare", enumType = "Plain", input = "null", accepted = true, numeric = value is null ? null : Numeric(value.Value), output = Serialize(value, typeof(Plain?)) });
}
catch (Exception error)
{
    results.Add(new { operation = "read", mode = "nullable-bare", enumType = "Plain", input = "null", accepted = false, error = Error(error) });
}
results.Add(new { operation = "write", mode = "concept", enumType = "Plain", numeric = (string?)null, output = Serialize(null, typeof(PlainConcept)) });
// Direct converter calls expose framework wrapping/null differences, not a separate wire contract.
foreach (var input in new[] { "null", "1.0", "1e0", "2147483648", "{}", "true" })
{
    foreach (var mode in new[] { "bare", "concept" })
    {
        try
        {
            var reader = new Utf8JsonReader(Encoding.UTF8.GetBytes(input));
            if (!reader.Read()) throw new InvalidOperationException("Empty reader input");
            object? value = mode == "bare"
                ? (object)new EnumConverter<Plain>().Read(ref reader, typeof(Plain), options)
                : (object?)new ConceptAsJsonConverter<PlainConcept>().Read(ref reader, typeof(PlainConcept), options);
            results.Add(new { operation = "direct-read", mode, input, accepted = true, display = value?.ToString() });
        }
        catch (Exception error) { results.Add(new { operation = "direct-read", mode, input, accepted = false, error = Error(error) }); }
    }
}
var fixture = new {
    source = new { repository = "Cratis/Fundamentals", commit = "d2accc4a79b6bcf2708213c97093ab5ba6c06381", package = "source extraction, not a published package" },
    runtime = new { framework = RuntimeInformation.FrameworkDescription, environmentVersion = Environment.Version.ToString(), architecture = RuntimeInformation.ProcessArchitecture.ToString(), os = RuntimeInformation.OSDescription, jsonAssembly = typeof(JsonSerializer).Assembly.FullName, jsonInformationalVersion = typeof(JsonSerializer).Assembly.GetCustomAttribute<AssemblyInformationalVersionAttribute>()?.InformationalVersion, culture = CultureInfo.CurrentCulture.Name },
    enumDefinitions = families.Select(f => new { name = f.Enum.Name, backing = Enum.GetUnderlyingType(f.Enum).Name, flags = f.Enum.IsDefined(typeof(FlagsAttribute), false), members = Enum.GetNames(f.Enum).Select(name => new { name, numeric = Numeric(Enum.Parse(f.Enum, name)) }) }),
    count = results.Count,
    cases = results
};
var json = JsonSerializer.Serialize(fixture, new JsonSerializerOptions { WriteIndented = true });
// Deliberately outside every observation catch: a swallowed stub exception
// invalidates the whole capture, never an individual expected failure.
if (Globals.AccessCount != 0)
    throw new InvalidOperationException($"Capture invalid: Globals accessed {Globals.AccessCount} times");
File.WriteAllText(args[0], json + "\n");
Console.WriteLine($"Captured {results.Count} .NET observations to {args[0]}; Globals accesses {Globals.AccessCount}; runtime {Environment.Version}, STJ {typeof(JsonSerializer).Assembly.FullName}");
object Serialize(object? value, Type type)
{
    try { return new { accepted = true, json = JsonSerializer.Serialize(value, type, options) }; }
    catch (Exception error) { return new { accepted = false, error = Error(error) }; }
}
static object Error(Exception error) => new { type = error.GetType().FullName, message = error.Message, innerType = error.InnerException?.GetType().FullName, innerMessage = error.InnerException?.Message };
static string Numeric(object value) => ((IFormattable)value).ToString("D", CultureInfo.InvariantCulture);

public enum Plain { Zero = 0, One = 1, Two = 2, Min = int.MinValue, Max = int.MaxValue }
public enum NoZero { One = 1 }
[Flags] public enum Bits { None = 0, A = 1, B = 2, AB = 3, C = 4 }
public enum ByteEnum : byte { Zero = 0, One = 1, Max = byte.MaxValue }
public enum UIntEnum : uint { Zero = 0, One = 1, High = 2147483648, Max = uint.MaxValue }
public enum LongEnum : long { Zero = 0, One = 1, Negative = -1, High = 2147483648, Low = -2147483649, Max = long.MaxValue }
public record PlainConcept(Plain Value) : ConceptAs<Plain>(Value);
public record NoZeroConcept(NoZero Value) : ConceptAs<NoZero>(Value);
public record BitsConcept(Bits Value) : ConceptAs<Bits>(Value);
public record ByteConcept(ByteEnum Value) : ConceptAs<ByteEnum>(Value);
public record UIntConcept(UIntEnum Value) : ConceptAs<UIntEnum>(Value);
public record LongConcept(LongEnum Value) : ConceptAs<LongEnum>(Value);
