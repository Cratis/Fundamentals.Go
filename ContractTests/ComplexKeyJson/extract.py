# Copyright (c) Cratis. All rights reserved.
# Licensed under the MIT license. See LICENSE file in the project root for full license information.
"""Extract pinned git objects into a new scratch directory; never fetch or mutate source."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import tempfile

PIN = 'd2accc4a79b6bcf2708213c97093ab5ba6c06381'
CS = ['Json/ComplexKeyDictionaryJsonConverterFactory.cs', 'Json/ConceptAsJsonConverter.cs',
      'Json/ConceptAsJsonConverterFactory.cs', 'Json/UnsupportedConceptValueType.cs',
      'Json/EnumerableConceptAsJsonConverter.cs', 'Json/EnumerableConceptAsJsonConverterFactory.cs',
      'Concepts/ConceptAs.cs', 'Concepts/ConceptMap.cs', 'Concepts/ConceptFactory.cs',
      'Concepts/ConceptExtensions.cs', 'Concepts/TypeIsNotAConcept.cs', 'Types/TypeConversion.cs',
      'Reflection/DictionaryExtensions.cs']


def extract(repo):
    def source(path):
        return subprocess.check_output(['git', '-C', str(repo), 'show', PIN + ':' + path])

    files = {'cs/source/' + p: source('Source/DotNET/Fundamentals/' + p) for p in CS}
    text = source('Source/DotNET/Fundamentals/Reflection/TypeExtensions.cs').decode('utf8')
    start = text.index('    /// <summary>\n    /// Get <see cref="ITypeInfo"')
    end = text.index('    /// <summary>', text.index('    public static ITypeInfo GetTypeInfoDetails', start))
    files['cs/source/Reflection/TypeExtensions.cs'] = (text[:start] + text[end:]).encode('utf8')
    pending = ['JsonSerializer.ts', 'fieldDecorator.ts', 'ConceptAs.ts', 'Guid.ts']
    seen = set()
    while pending:
        path = pending.pop()
        if path in seen:
            continue
        seen.add(path)
        data = source('Source/JavaScript/' + path)
        files['js/source/' + path] = data
        for dep in re.findall(r"(?:from\s+|import\s*)['\"]([^'\"]+)['\"]", data.decode('utf8')):
            if not dep.startswith('.'):
                raise ValueError('External dependency: ' + dep)
            rel = os.path.normpath(str(Path(path).parent / dep))
            pending.append(rel + ('/index.ts' if rel in ('json', 'geospatial') else '.ts'))
    files['LICENSE'] = source('LICENSE')
    if len(CS) != 13 or len(seen) != 34:
        raise ValueError('Pinned source inventory changed')
    return files


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--source-repo', required=True, type=Path)
    parser.add_argument('--output', required=True, type=Path)
    args = parser.parse_args()
    harness = Path(__file__).resolve().parent
    output = args.output.resolve()
    allowed = [harness.parents[1] / '.ai-work', Path(tempfile.gettempdir()).resolve()]
    if not any(output.is_relative_to(p) and output != p for p in allowed):
        parser.error('--output must be below this checkout .ai-work or system temporary directory')
    if output.exists() and any(output.iterdir()):
        parser.error('--output must be new or empty; never overwrite evidence')
    files = extract(args.source_repo)
    expected = json.loads((harness.parents[1] / 'testdata/complex-key-contract/manifest.json').read_text())['sourceFiles']
    actual = {p: hashlib.sha256(data).hexdigest() for p, data in sorted(files.items())}
    if actual != expected:
        raise ValueError('Pinned extraction differs from reviewed source inventory')
    for name in ['cs/Program.cs', 'cs/Probe.csproj', 'cs/UnreachableGlobals.cs', 'js/probe.ts', 'js/tsconfig.json']:
        files[name] = (harness / name).read_bytes()
    files['inputs.json'] = (harness.parents[1] / 'testdata/complex-key-contract/inputs.json').read_bytes()
    for name, data in files.items():
        target = output / name
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_bytes(data)
    print('Extracted 13 exact C# + 1 reduced reflection + 34 exact TS + LICENSE; no packages; output:', output)


if __name__ == '__main__':
    main()
