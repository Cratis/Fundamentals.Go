using System.Globalization;
using System.Reflection;
using System.Runtime.InteropServices;
using System.Security.Cryptography;
using System.Text;
using System.Text.Json;

// Inputs are literal/structural mutations of the issue-28 plan, never Go outputs.
// Predictions are annotations, not filters or assertions on runtime behavior.
internal static class Program
{
    const string D = "00112233-4455-6677-8899-aabbccddeeff";
    const string N = "00112233445566778899aabbccddeeff";
    const string X = "{0x00112233,0x4455,0x6677,{0x88,0x99,0xaa,0xbb,0xcc,0xdd,0xee,0xff}}";
    const string Vmr = "95017c711e6afc1085133d440e42b4bd78155701";
    const string RuntimeSource = "4271d88e0aebf3d04f188f1334c2220d80555ef6";
    static readonly string[] Tokens = ["0x00112233", "0x4455", "0x6677", "0x88", "0x99", "0xaa", "0xbb", "0xcc", "0xdd", "0xee", "0xff"];
    static readonly char[] Whitespace = [.. Enumerable.Range(9, 5).Select(n => (char)n), ' ', '\u0085', '\u00a0', '\u1680', .. Enumerable.Range(0x2000, 11).Select(n => (char)n), '\u2028', '\u2029', '\u202f', '\u205f', '\u3000'];
    static readonly JsonSerializerOptions Options = new() { WriteIndented = true, PropertyNamingPolicy = JsonNamingPolicy.CamelCase };
    static readonly List<Case> Cases = [];
    static readonly HashSet<string> Ids = new(StringComparer.Ordinal);

    static int Main(string[] args)
    {
        if (args.Length != 1) throw new ArgumentException("Supply one NEW capture JSON path.");
        CultureInfo.CurrentCulture = CultureInfo.InvariantCulture;
        CultureInfo.CurrentUICulture = CultureInfo.InvariantCulture;
        if (Environment.Version.ToString() != "10.0.12") throw new InvalidOperationException("Wrong runtime.");
        var sdk = Assembly.GetExecutingAssembly().GetCustomAttributes<AssemblyMetadataAttribute>().Single(a => a.Key == "ProbeSdkVersion").Value;
        if (sdk != "10.0.401") throw new InvalidOperationException("Wrong SDK.");
        if (Whitespace.Length != 25 || Whitespace.Distinct().Count() != 25 || MakeX(Tokens) != X) throw new InvalidOperationException("Broken generator constants.");
        Generate();
        var observations = Cases.Select(Observe).ToArray();
        var diagnostics = new[] {
            new Case("diagnostic/null-string", null, null, null),
            new Case("diagnostic/unpaired-high-only", "\ud800", null, null),
            new Case("diagnostic/unpaired-low-only", "\udfff", null, null),
            new Case("diagnostic/unpaired-high-before-D", "\ud800" + D, null, null),
            new Case("diagnostic/unpaired-low-after-X", X + "\udfff", null, null),
            new Case("diagnostic/unpaired-high-in-D", D[..35] + "\ud800", null, null)
        }.Select(c => {
            if (!Ids.Add(c.Id)) throw new InvalidOperationException("Duplicate diagnostic ID.");
            // JSON cannot faithfully represent unpaired UTF-16. Exact code units remain authoritative.
            var observed = Observe(c);
            return new { observed.Id, inputKind = c.Input is null ? "null-string" : "unpaired-utf16", inputUtf16Hex = Utf16Hex(c.Input), observed.Parse, observed.TryParse, observed.BuiltInJson, note = "Outside Go valid-UTF8 input contract; no Go equivalence. JSON string serialization may replace unpaired surrogates; inspect serializedJson." };
        }).ToArray();
        var rawJson = new (string Id, string Input)[] {
            ("json/escaped-all-D-characters", "\"" + string.Concat(D.Select(c => "\\u" + ((int)c).ToString("x4"))) + "\""),
            ("json/escaped-hyphens", "\"" + D.Replace("-", "\\u002d") + "\""),
            ("json/escaped-hex-digits", "\"" + D.Replace("a", "\\u0061") + "\""),
            ("json/outside-whitespace", " \t\r\n\"" + D + "\" \t\r\n"),
            ("json/null", "null"), ("json/number-zero", "0"), ("json/boolean-true", "true"),
            ("json/object", "{}"), ("json/array", "[]"),
            ("json/raw-unescaped-NUL", "\"" + D + "\0\""),
            ("json/escaped-NUL", "\"" + D + "\\u0000\""),
            ("json/second-token", "\"" + D + "\" true")
        }.Select(c => {
            if (!Ids.Add(c.Id)) throw new InvalidOperationException("Duplicate raw JSON ID.");
            return new { id = c.Id, rawJson = c.Input, rawJsonUtf8Hex = Convert.ToHexStringLower(Encoding.UTF8.GetBytes(c.Input)), builtInJson = ReadJson(c.Input) };
        }).ToArray();
        var goSeeds = new[] {
            (Id: "go-invalid-utf8/ff", Hex: "ff"),
            (Id: "go-invalid-utf8/c0-af", Hex: "c0af"),
            (Id: "go-invalid-utf8/ff-before-D", Hex: "ff" + Convert.ToHexStringLower(Encoding.UTF8.GetBytes(D))),
            (Id: "go-invalid-utf8/c0-af-in-D", Hex: Convert.ToHexStringLower(Encoding.UTF8.GetBytes(D[..4])) + "c0af" + Convert.ToHexStringLower(Encoding.UTF8.GetBytes(D[4..])))
        }.Select(c => new { id = c.Id, inputBytesHex = c.Hex, disposition = "Go-only rejection seed; not passed to .NET; no runtime-equivalent string asserted." }).ToArray();
        foreach (var c in goSeeds) if (!Ids.Add(c.id)) throw new InvalidOperationException("Duplicate seed ID.");
        var runtimeDirectory = RuntimeEnvironment.GetRuntimeDirectory();
        var versionFile = File.ReadAllText(Path.Combine(runtimeDirectory, ".version"));
        if (!versionFile.Contains(Vmr, StringComparison.Ordinal)) throw new InvalidOperationException("Runtime .version does not match source pin.");
        var sourceHashes = new[] { "Program.cs", "GuidParseProbe.csproj", "global.json", "NuGet.Config" }.ToDictionary(p => p, p => Hash(File.ReadAllBytes(p)));
        var capture = new {
            schemaVersion = 1,
            metadata = new {
                target = ".NET 10.0.12 Guid.Parse(string)/Guid.TryParse(string)",
                sdkVersion = sdk, runtimeVersion = Environment.Version.ToString(), frameworkDescription = RuntimeInformation.FrameworkDescription,
                osDescription = RuntimeInformation.OSDescription, osArchitecture = RuntimeInformation.OSArchitecture.ToString(), processArchitecture = RuntimeInformation.ProcessArchitecture.ToString(),
                coreLib = AssemblyInfo(typeof(Guid).Assembly), systemTextJson = AssemblyInfo(typeof(JsonSerializer).Assembly),
                culture = CultureInfo.CurrentCulture.Name, uiCulture = CultureInfo.CurrentUICulture.Name, invariantGlobalization = AppContext.TryGetSwitch("System.Globalization.Invariant", out var invariant) && invariant,
                runtimeFrameworkVersion = "10.0.12", rollForward = "Disable", runtimeVersionFile = versionFile,
                vmrSourceCommit = Vmr, officialRuntimeGuidSourceCommit = RuntimeSource,
                sourcePinBasis = "Prior source plan verified Guid and JSON reader files against these pins; this capture additionally checks installed runtime .version. No Cratis packages loaded or equivalence claimed.",
                sourceHashes,
                inputHashAlgorithm = "SHA-256 of UTF-8 compact JSON array of {id,input} in corpus order; valid Unicode corpus only",
                inputSha256 = Hash(JsonSerializer.SerializeToUtf8Bytes(Cases.Select(c => new { id = c.Id, input = c.Input }))),
                // Portable profile: preserve the historical plan digest as provenance,
                // not a dependency on a maintainer's private work-record directory.
                captureProfile = "portable-guid-parse-v1",
                originalCaptureSha256 = "e441b368b467b81005c885e7a2cab04ba407354ff2a564a33c56c0315b21057c",
                planSha256 = "3f547f5a2e261448f20324b9d61fc1e00eca05f70be645ff3ed38e651dc89477",
                exceptionPolicy = "Parse catches only FormatException, OverflowException, ArgumentNullException; JSON catches only JsonException. Other failures abort capture. Predictions never force observations."
            },
            counts = new { strings = observations.Length, dotNetDiagnostics = diagnostics.Length, rawJson = rawJson.Length, goOnlySeeds = goSeeds.Length, allIds = Ids.Count },
            whitespaceCodePoints = Whitespace.Select(c => $"U+{(int)c:X4}"),
            cases = observations, dotNetDiagnostics = diagnostics, rawJsonControls = rawJson, goInvalidUtf8Seeds = goSeeds
        };
        var bytes = JsonSerializer.SerializeToUtf8Bytes(capture, Options);
        // Validate the actual JSON before it can become the immutable observation file.
        using (var document = JsonDocument.Parse(bytes)) Validate(document.RootElement, observations.Length, Ids.Count);
        using (var file = new FileStream(args[0], FileMode.CreateNew, FileAccess.Write, FileShare.None)) file.Write(bytes);
        Console.WriteLine($"Captured {observations.Length} strings, {diagnostics.Length} .NET diagnostics, {rawJson.Length} raw JSON controls, {goSeeds.Length} Go seeds; SHA256={Hash(bytes)}");
        foreach (var o in observations.Where(o => o.Prediction.Accepted != o.Parse.Accepted || (o.Prediction.CanonicalD is not null && o.Prediction.CanonicalD != o.Parse.CanonicalD)))
            Console.WriteLine($"PREDICTION MISMATCH {o.Id}: expected {o.Prediction.Accepted}/{o.Prediction.CanonicalD}, observed {o.Parse.Accepted}/{o.Parse.CanonicalD}/{o.Parse.ExceptionType}");
        return 0;
    }

    static void Generate()
    {
        var forms = new Dictionary<string, string> { ["D"] = D, ["N"] = N, ["B"] = "{" + D + "}", ["P"] = "(" + D + ")", ["X"] = X };
        foreach (var (format, input) in forms)
        {
            Add($"canonical/{format}/asymmetric", input, true, D);
            Add($"canonical/{format}/uppercase", input.ToUpperInvariant(), true, D);
            // Each a-f occurrence is uppercased independently, retaining its absolute position.
            for (var i = 0; i < input.Length; i++) if (input[i] is >= 'a' and <= 'f')
                Add($"case/{format}/position-{i:D2}-{input[i]}", input[..i] + char.ToUpperInvariant(input[i]) + input[(i + 1)..], true, D);
            foreach (var ws in Whitespace)
            {
                Add($"whitespace/{format}/U+{(int)ws:X4}/leading", ws + input, true, D);
                Add($"whitespace/{format}/U+{(int)ws:X4}/trailing", input + ws, true, D);
                Add($"whitespace/{format}/U+{(int)ws:X4}/both", ws + input + ws, true, D);
            }
            foreach (var nonws in new[] { '\0', '\u001c', '\u180e', '\u200b', '\ufeff' })
            {
                Add($"non-whitespace/{format}/U+{(int)nonws:X4}/leading", nonws + input, false);
                Add($"non-whitespace/{format}/U+{(int)nonws:X4}/trailing", input + nonws, false);
                Add($"non-whitespace/{format}/U+{(int)nonws:X4}/both", nonws + input + nonws, false);
            }
        }
        foreach (var digit in new[] { '0', 'f' })
        {
            var n = new string(digit, 32);
            var d = $"{n[..8]}-{n[8..12]}-{n[12..16]}-{n[16..20]}-{n[20..]}";
            var x = MakeX(["0x" + new string(digit, 8), "0x" + new string(digit, 4), "0x" + new string(digit, 4), .. Enumerable.Repeat("0x" + new string(digit, 2), 8)]);
            foreach (var (format, input) in new Dictionary<string, string> { ["D"] = d, ["N"] = n, ["B"] = "{" + d + "}", ["P"] = "(" + d + ")", ["X"] = x })
                Add($"canonical/{format}/{(digit == '0' ? "zero" : "max")}", input, true, d);
        }
        foreach (var ws in Whitespace) Add($"X/internal-whitespace/U+{(int)ws:X4}/every-gap", string.Join(ws.ToString(), X.Select(c => c.ToString())), true, D);
        string[] positiveTokens = ["0x0", "0x1", "0x0000000001", "0XAb", "0x+1", "0x0x1", "0x+0X1", "0x+", "0x0x", "0x+0x"];
        string[] negativeTokens = ["1", "0x", "0x-1", "-0x1", "+0x1", "0x++1", "0x0x+1", "0x0x0x1", "0xg", "0x1_0", "0x1.0", "0x١"];
        for (var field = 0; field < 11; field++)
        {
            foreach (var token in positiveTokens) TokenCase(field, token, "legacy-positive", true);
            foreach (var token in negativeTokens) TokenCase(field, token, "negative", false);
            TokenCase(field, "0x" + new string('0', 64) + Tokens[field][2..], "64-leading-zeroes", true);
            if (field < 3)
            {
                foreach (var token in new[] { "0xffffffff", "0x00000000ffffffff" }) TokenCase(field, token, "uint32-max", true);
                TokenCase(field, "0x100000000", "uint32-overflow", false);
            }
            if (field is 1 or 2) foreach (var token in new[] { "0xffff", "0x10000", "0x12345678" }) TokenCase(field, token, "uint16-truncation", true);
            if (field >= 3)
            {
                foreach (var token in new[] { "0xff", "0x000000ff" }) TokenCase(field, token, "byte-max", true);
                foreach (var token in new[] { "0x100", "0xffff", "0x100000000" }) TokenCase(field, token, "byte-overflow", false);
            }
        }
        var fallback = new (string Name, string Input, string Canonical)[] {
            ("plus", "+ddddddd-+ddd-+ddd-+ddd-+ddddddddddd", "0ddddddd-0ddd-0ddd-0ddd-0ddddddddddd"),
            ("prefix", "0xdddddd-0xdd-0xdd-0xdd-0xdddddddddd", "00dddddd-00dd-00dd-00dd-00dddddddddd"),
            ("plus-prefix", "+0Xddddd-+0Xd-+0Xd-+0Xd-+0Xddddddddd", "000ddddd-000d-000d-000d-000ddddddddd"),
            ("conditional-NUL-tail", "+0112233-4455-6677-8899-aabbccddeef\0", "00112233-4455-6677-8899-aabb0ccddeef")
        };
        foreach (var c in fallback)
        {
            Add($"D-fallback/{c.Name}/D", c.Input, true, c.Canonical);
            Add($"D-fallback/{c.Name}/B", "{" + c.Input + "}", true, c.Canonical);
            Add($"D-fallback/{c.Name}/P", "(" + c.Input + ")", true, c.Canonical);
        }
        foreach (var (name, input) in new (string, string)[] {
            ("empty", ""), ("only-whitespace", " \t\r\n"), ("word", "invalid"),
            ("N-short", N[..^1]), ("N-long", N + "0"), ("D-bad-hex", D[..^1] + "g"), ("D-appended-punctuation", D + "!"),
            ("N-inner-space", N.Insert(4, " ")), ("D-inner-space", D.Insert(4, " ")), ("B-inner-spaces", "{ " + D + " }"), ("P-inner-spaces", "( " + D + " )"),
            ("URN-lower", "urn:uuid:" + D), ("URN-upper", "URN:UUID:" + D),
            ("N-braces", "{" + N + "}"), ("N-parentheses", "(" + N + ")"), ("D-brackets", "[" + D + "]"), ("D-angle-brackets", "<" + D + ">"),
            ("D-punctuation", "!" + D + "?"), ("mismatch-brace-paren", "{" + D + ")"), ("mismatch-paren-brace", "(" + D + "}"), ("nested-braces", "{{" + D + "}}"),
            ("B-missing-open", D + "}"), ("B-missing-close", "{" + D), ("P-missing-open", D + ")"), ("P-missing-close", "(" + D),
            ("D-extra-prefix", "0x00112233-4455-6677-8899-aabbccddeeff"), ("D-negative", "-0112233-4455-6677-8899-aabbccddeeff"),
            ("D-split-tail-plus", "00112233-4455-6677-8899-aabb+cddeeff"), ("D-split-tail-prefix", "00112233-4455-6677-8899-aabb0xddeeff"),
            ("D-NUL-no-fallback-trigger", "00112233-4455-6677-8899-aabbccddeef\0")
        }) Add("invalid/" + name, input, false);
        foreach (var index in new[] { 8, 13, 18, 23 })
        {
            Add($"invalid/D-hyphen-{index}-removed", D.Remove(index, 1), false);
            Add($"invalid/D-hyphen-{index}-underscore", D[..index] + "_" + D[(index + 1)..], false);
        }
        Add("invalid/X-seven-bytes", MakeX(Tokens[..^1]), false);
        Add("invalid/X-nine-bytes", MakeX([.. Tokens, "0x00"]), false);
        Add("invalid/X-trailing-comma", X[..^2] + ",}}", false);
        Add("invalid/X-doubled-first-comma", X.Replace("0x00112233,", "0x00112233,,", StringComparison.Ordinal), false);
        for (var i = 0; i < X.Length; i++) if (X[i] is '{' or '}') Add($"invalid/X-brace-{i}-removed", X.Remove(i, 1), false);
        Add("invalid/X-parentheses", "(" + X[1..^1] + ")", false);
        Add("invalid/X-appended-punctuation", X + "!", false);
    }

    static void TokenCase(int field, string token, string group, bool accepted)
    {
        var tokens = Tokens.ToArray();
        tokens[field] = token;
        Add($"X/field-{field + 1:D2}/{group}/{token}", MakeX(tokens), accepted);
    }
    static string MakeX(string[] tokens) => "{" + string.Join(',', tokens[..3]) + ",{" + string.Join(',', tokens[3..]) + "}}";
    static void Add(string id, string input, bool accepted, string? canonical = null)
    {
        if (!Ids.Add(id)) throw new InvalidOperationException("Duplicate case ID: " + id);
        _ = new UTF8Encoding(false, true).GetBytes(input); // generator must not silently replace malformed Unicode
        Cases.Add(new(id, input, accepted, canonical));
    }
    static Observation Observe(Case c)
    {
        Result parse;
        try { parse = Success(Guid.Parse(c.Input!)); }
        catch (Exception e) when (e is FormatException or OverflowException or ArgumentNullException) { parse = Failure(e); }
        var accepted = Guid.TryParse(c.Input, out var value);
        var tryParse = new TryResult(accepted, value.ToString("D"), Convert.ToHexStringLower(value.ToByteArray(bigEndian: true)));
        if (accepted != parse.Accepted || (accepted && (tryParse.CanonicalD != parse.CanonicalD || tryParse.RfcBytesHex != parse.RfcBytesHex)) || (!accepted && value != Guid.Empty))
            throw new InvalidOperationException("Parse/TryParse disagreement: " + c.Id);
        var json = JsonSerializer.Serialize(c.Input);
        return new(c.Id, c.Input, c.Input is not null && IsWellFormed(c.Input) ? Convert.ToHexStringLower(new UTF8Encoding(false, true).GetBytes(c.Input)) : null,
            new(c.Predicted, c.Canonical), parse, tryParse, new(json, ReadJson(json)));
    }
    static bool IsWellFormed(string input)
    {
        try { _ = new UTF8Encoding(false, true).GetByteCount(input); return true; }
        catch (EncoderFallbackException) { return false; }
    }
    static Result ReadJson(string json)
    {
        try { return Success(JsonSerializer.Deserialize<Guid>(json)); }
        catch (JsonException e) { return Failure(e); }
    }
    static Result Success(Guid value) => new(true, value.ToString("D"), Convert.ToHexStringLower(value.ToByteArray(bigEndian: true)), null, null);
    static Result Failure(Exception e) => new(false, null, null, e.GetType().FullName!, e.InnerException?.GetType().FullName);
    static string? Utf16Hex(string? input) => input is null ? null : string.Join(' ', input.Select(c => ((int)c).ToString("x4")));
    static string Hash(byte[] bytes) => Convert.ToHexStringLower(SHA256.HashData(bytes));
    static object AssemblyInfo(Assembly a) => new { name = a.GetName().Name, assemblyVersion = a.GetName().Version!.ToString(), informationalVersion = a.GetCustomAttribute<AssemblyInformationalVersionAttribute>()!.InformationalVersion, fileVersion = a.GetCustomAttribute<AssemblyFileVersionAttribute>()!.Version };
    static void Validate(JsonElement root, int count, int allIds)
    {
        if (root.GetProperty("schemaVersion").GetInt32() != 1 || root.GetProperty("cases").GetArrayLength() != count || count < 800) throw new InvalidOperationException("Capture schema/count failed.");
        var ids = new HashSet<string>(StringComparer.Ordinal);
        foreach (var array in new[] { "cases", "dotNetDiagnostics", "rawJsonControls", "goInvalidUtf8Seeds" })
            foreach (var row in root.GetProperty(array).EnumerateArray())
                if (!ids.Add(row.GetProperty("id").GetString()!)) throw new InvalidOperationException("Serialized duplicate ID.");
        if (ids.Count != allIds) throw new InvalidOperationException("Serialized count mismatch.");
        foreach (var row in root.GetProperty("cases").EnumerateArray())
        {
            _ = row.GetProperty("input").GetString() ?? throw new InvalidOperationException("Missing input.");
            _ = Convert.FromHexString(row.GetProperty("inputUtf8Hex").GetString()!);
            foreach (var result in new[] { row.GetProperty("parse"), row.GetProperty("tryParse"), row.GetProperty("builtInJson").GetProperty("result") })
            {
                var accepted = result.GetProperty("accepted").GetBoolean();
                if (accepted || result.TryGetProperty("exceptionType", out _) == false)
                {
                    if (result.GetProperty("canonicalD").GetString()!.Length != 36 || Convert.FromHexString(result.GetProperty("rfcBytesHex").GetString()!).Length != 16) throw new InvalidOperationException("Bad value shape.");
                }
                else if (result.GetProperty("exceptionType").ValueKind != JsonValueKind.String) throw new InvalidOperationException("Missing failure type.");
            }
        }
    }
    sealed record Case(string Id, string? Input, bool? Predicted, string? Canonical);
    sealed record Prediction(bool? Accepted, string? CanonicalD);
    sealed record Result(bool Accepted, string? CanonicalD, string? RfcBytesHex, string? ExceptionType, string? InnerExceptionType);
    sealed record TryResult(bool Accepted, string CanonicalD, string RfcBytesHex);
    sealed record JsonResult(string SerializedJson, Result Result);
    sealed record Observation(string Id, string? Input, string? InputUtf8Hex, Prediction Prediction, Result Parse, TryResult TryParse, JsonResult BuiltInJson);
}
