// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.
using System.Collections;
using System.Text.Json;
using Cratis.Concepts;
using Cratis.Json;

var root = Path.GetFullPath(args[0]);
var options = new JsonSerializerOptions
{
    PropertyNamingPolicy = JsonNamingPolicy.CamelCase,
    Converters = { new ComplexKeyDictionaryJsonConverterFactory(), new EnumerableConceptAsJsonConverterFactory(), new ConceptAsJsonConverterFactory() }
};
var outputOptions = new JsonSerializerOptions { WriteIndented = true };
var kinds = new Dictionary<string, Type>
{
    ["string"] = typeof(string), ["stringConcept"] = typeof(TextKey), ["guid"] = typeof(Guid),
    ["guidConcept"] = typeof(IdKey), ["composite"] = typeof(Composite), ["compositeObjectValue"] = typeof(Composite)
};
var observations = new List<object>();
object Failure(string stage, Exception ex) => new { stage, status = "rejected", exception = ex.GetType().FullName, innerException = ex.InnerException?.GetType().FullName, message = ex.Message };
object Observe(string stage, Func<object?> action)
{
    try { return new { stage, status = "accepted", value = action() }; }
    catch (Exception ex) { return Failure(stage, ex); }
}
Type ValueType(string kind) => kind == "compositeObjectValue" ? typeof(Payload) : typeof(int);
Type MapType(string kind) => typeof(IDictionary<,>).MakeGenericType(kinds[kind], ValueType(kind));
Type EnvelopeType(string kind) => typeof(Envelope<,>).MakeGenericType(kinds[kind], ValueType(kind));
object? Describe(object? value) => value switch
{
    null => null,
    TextKey text => new { type = "TextKey", value = text.Value },
    IdKey id => new { type = "IdKey", value = id.Value.ToString() },
    Guid id => new { type = "Guid", value = id.ToString() },
    Composite key => new { type = "Composite", id = key.Id.ToString(), name = key.Name },
    _ => value
};
object DescribeMap(object? value)
{
    if (value is null) return new { state = "null" };
    var map = (IDictionary)value;
    var entries = new List<object>();
    foreach (DictionaryEntry entry in map)
        entries.Add(new { key = Describe(entry.Key), value = Describe(entry.Value) });
    return new { state = "map", count = map.Count, entries };
}
void Read(string id, string kind, string raw, string origin, string sourceCapture, string sourceCaseId)
{
    object? returned = null;
    var read = Observe("read", () =>
    {
        returned = JsonSerializer.Deserialize(raw, EnvelopeType(kind), options);
        if (returned is null) return new { envelope = "null" };
        return DescribeMap(EnvelopeType(kind).GetProperty("Map")!.GetValue(returned));
    });
    var rewrite = returned is null ? (object)new { stage = "rewrite", status = "not-attempted", reason = "no-returned-object" } : Observe("rewrite", () => JsonSerializer.Serialize(returned, EnvelopeType(kind), options));
    observations.Add(new { id, kind, operation = "read", origin, sourceCapture, sourceCaseId, input = raw, read, rewrite, returnedObject = returned is not null });
}
void Write(string id, string kind, object? key, object? value, bool indented = false)
{
    var writeOptions = new JsonSerializerOptions(options) { WriteIndented = indented };
    var map = (IDictionary)Activator.CreateInstance(typeof(Dictionary<,>).MakeGenericType(kinds[kind], ValueType(kind)))!;
    var construction = Observe("construction", () => { map.Add(key!, value); return DescribeMap(map); });
    var write = Observe("write", () => JsonSerializer.Serialize(map, MapType(kind), writeOptions));
    observations.Add(new { id, kind, operation = "write", sourceCapture = (string?)null, sourceCaseId = (string?)null, inputKey = Describe(key), inputValue = Describe(value), construction, write, indented });
}
if (args.Length > 1 && args[1] == "cross")
{
    using var js = JsonDocument.Parse(File.ReadAllText(Path.Combine(root, "javascript.json")));
    foreach (var row in js.RootElement.GetProperty("observations").EnumerateArray())
    {
        if (row.GetProperty("operation").GetString() != "write") continue;
        var write = row.GetProperty("write");
        if (write.GetProperty("status").GetString() != "accepted") continue;
        Read("js/" + row.GetProperty("id").GetString(), row.GetProperty("kind").GetString()!, write.GetProperty("value").GetString()!, "actual-js-write", "javascript.json", row.GetProperty("id").GetString()!);
    }
}
else
{
    using var cases = JsonDocument.Parse(File.ReadAllText(Path.Combine(root, "inputs.json")));
    foreach (var row in cases.RootElement.EnumerateArray())
        Read(row.GetProperty("id").GetString()!, row.GetProperty("kind").GetString()!, row.GetProperty("input").GetString()!, "authored-stimulus", "inputs.json", row.GetProperty("id").GetString()!);
    var guid = Guid.Parse("e094c582-9d91-4df0-b795-8d39fce089da");
    foreach (var text in new[] { "plain", "a\"b", "a\\b", "é雪😀<>&'\"", "" })
    {
        var n = observations.Count;
        Write("string/write/" + n, "string", text, 42);
        Write("stringConcept/write/" + n, "stringConcept", new TextKey(text), 42);
    }
    Write("guid/write", "guid", guid, 42);
    Write("guidConcept/write", "guidConcept", new IdKey(guid), 42);
    Write("guidConcept/write-zero", "guidConcept", new IdKey(Guid.Empty), 0);
    Write("composite/write", "composite", new Composite(guid, "tenant-a"), 42);
    Write("composite/write-escaped", "composite", new Composite(guid, "a\"\\é雪😀<>&"), 42);
    Write("composite/write-indented", "composite", new Composite(guid, "tenant-a"), 42, true);
    Write("composite/write-null-member", "composite", new Composite(guid, null), 42);
    Write("compositeObjectValue/write", "compositeObjectValue", new Composite(guid, "tenant-a"), new Payload(42, "answer"));
    Write("compositeObjectValue/write-null", "compositeObjectValue", new Composite(guid, "tenant-a"), null);
}
// Fail outside observation catches; unexpected dependency access invalidates ALL capture evidence.
if (Globals.AccessCount != 0) throw new InvalidOperationException($"Unexpected global accesses: {Globals.AccessCount}");
var ids = observations.Select(row => (string)row.GetType().GetProperty("id")!.GetValue(row)!).ToArray();
if (ids.Length == 0 || ids.Distinct().Count() != ids.Length) throw new InvalidOperationException("Empty or duplicate observations");
var factory = new ComplexKeyDictionaryJsonConverterFactory();
var admissionTypes = new[] { typeof(Dictionary<string, int>), typeof(Dictionary<Guid, int>), typeof(Dictionary<int, int>), typeof(Dictionary<decimal, int>), typeof(Dictionary<Uri, int>), typeof(Dictionary<DateTime, int>), typeof(Dictionary<TextKey, int>), typeof(IDictionary<TextKey, int>), typeof(IReadOnlyDictionary<TextKey, int>), typeof(Dictionary<Composite, int>), typeof(Hashtable) };
var result = new
{
    schemaVersion = 1, sourceRevision = "d2accc4a79b6bcf2708213c97093ab5ba6c06381", runtime = System.Runtime.InteropServices.RuntimeInformation.FrameworkDescription,
    systemTextJson = typeof(JsonSerializer).Assembly.GetCustomAttributesData().First(a => a.AttributeType.Name == "AssemblyInformationalVersionAttribute").ConstructorArguments[0].Value,
    profile = "CamelCase/default encoder/compact; ComplexKey, EnumerableConceptAs, ConceptAs factories; not Globals", globalStubAccesses = Globals.AccessCount,
    count = ids.Length, factoryAdmission = admissionTypes.Select(type => new { type = type.ToString(), admitted = factory.CanConvert(type) }), observations
};
var output = Path.Combine(root, args.Length > 1 ? "csharp-cross.json" : "csharp.json");
File.WriteAllText(output, JsonSerializer.Serialize(result, outputOptions) + "\n");
Console.WriteLine($"Captured {ids.Length} unique C# observations; {Globals.AccessCount} global accesses; {output}");

public record TextKey(string Value) : ConceptAs<string>(Value);
public record IdKey(Guid Value) : ConceptAs<Guid>(Value);
public record Composite(Guid Id, string? Name);
public record Payload(int Number, string? Label);
public class Envelope<TKey, TValue> where TKey : notnull
{
    public IDictionary<TKey, TValue>? Map { get; set; } = new Dictionary<TKey, TValue>();
}
