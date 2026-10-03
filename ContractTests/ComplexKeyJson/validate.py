# Copyright (c) Cratis. All rights reserved.
# Licensed under the MIT license. See LICENSE file in the project root for full license information.
"""Dataset invariants, not a JSON Schema engine or complex-key implementation."""
import json
from pathlib import Path

FIXTURES = Path(__file__).resolve().parents[2] / 'testdata/complex-key-contract'
PIN = 'd2accc4a79b6bcf2708213c97093ab5ba6c06381'
METADATA = [
    {'kind': kind, 'field': 'map', 'type': 'ValueMap', 'genericArguments': args}
    for kind, args in [('string', ['String', 'Number']), ('stringConcept', ['TextKey', 'Number']),
                       ('guid', ['Guid', 'Number']), ('guidConcept', ['IdKey', 'Number']),
                       ('composite', ['Composite', 'Number']), ('compositeObjectValue', ['Composite', 'Payload'])]
]


def json_equal(a, b):
    if type(a) is not type(b):
        return False
    if isinstance(a, dict):
        return a.keys() == b.keys() and all(json_equal(a[k], b[k]) for k in a)
    if isinstance(a, list):
        return len(a) == len(b) and all(json_equal(x, y) for x, y in zip(a, b))
    return a == b


def require(condition, message):
    if not condition:
        raise ValueError(message)


def load(root):
    return {name: json.loads((root / name).read_text(encoding='utf8'))
            for name in ['inputs.json', 'csharp.json', 'javascript.json', 'csharp-cross.json']}


def outcome(value, stage, lookup=False):
    require(type(value) is dict, 'missing outcome ' + stage)
    extra = {'queryMode', 'lookupKey'} if lookup else set()
    if lookup:
        require(value.get('queryMode') in ('fixed-key', 'original-key') and 'lookupKey' in value, 'lookup metadata')
    require(value.get('stage') == stage, 'outcome stage')
    status = value.get('status')
    if status == 'accepted':
        keys = {'stage', 'status', 'value'}
    elif status == 'rejected':
        keys = {'stage', 'status', 'exception', 'innerException', 'message'}
        require(type(value.get('exception')) is str and value['exception'] != '', 'error category')
        require(type(value.get('message')) is str, 'diagnostic message')
        require(value.get('innerException') is None or type(value['innerException']) is str, 'inner category')
    elif status == 'not-attempted':
        keys = {'stage', 'status', 'reason'}
        require(type(value.get('reason')) is str and value['reason'] != '', 'not-attempted reason')
    else:
        raise ValueError('unknown outcome status')
    require(set(value) == keys | extra, 'outcome fields/presence ' + stage)


def validate(data, manifest):
    index = {}
    for name, capture in data.items():
        rows = capture if name == 'inputs.json' else capture['observations']
        require(type(rows) is list, 'rows must be array')
        inventory = manifest['datasets'][name]
        require(len(rows) == inventory['count'], 'record count ' + name)
        ids = [r.get('id') for r in rows]
        require(json_equal(ids, inventory['ids']) and len(set(ids)) == len(ids), 'exact IDs ' + name)
        index[name] = {r['id']: r for r in rows}
        if name == 'inputs.json':
            for row in rows:
                require(set(row) == {'id', 'kind', 'input'} and type(row['input']) is str, 'input fields')
                require(row['kind'] in manifest['declarations'], 'input kind')
            continue
        require(type(capture.get('schemaVersion')) is int and capture['schemaVersion'] == 1, 'schema version')
        require(capture.get('sourceRevision') == PIN, 'source revision')
        require(type(capture.get('count')) is int and capture['count'] == len(rows), 'typed capture count')
        require(type(capture.get('profile')) is str and capture['profile'] != '', 'profile')
        if name == 'javascript.json':
            require(json_equal(capture.get('metadata'), METADATA), 'typed field metadata')
            require(type(capture.get('typescript')) is str and capture['typescript'] != '', 'compiler version')
        else:
            require(type(capture.get('globalStubAccesses')) is int and capture['globalStubAccesses'] == 0, 'zero Globals accesses')
            require(len(capture.get('factoryAdmission', [])) == 11, 'admission inventory')
            for a in capture['factoryAdmission']:
                require(set(a) == {'type', 'admitted'} and type(a['type']) is str and type(a['admitted']) is bool, 'typed admission')
        operations = {}
        lookups = {}
        for row in rows:
            op = row.get('operation')
            operations[op] = operations.get(op, 0) + 1
            require('sourceCapture' in row and 'sourceCaseId' in row, 'source linkage presence')
            if op == 'equality':
                require(row['sourceCapture'] is None and row['sourceCaseId'] is None, 'equality source')
                require(type(row.get('count')) is int, 'equality count')
                continue
            require(row.get('kind') in manifest['declarations'], 'declared kind')
            if op == 'write':
                require(row['sourceCapture'] is None and row['sourceCaseId'] is None, 'write source')
                require('inputKey' in row and 'inputValue' in row, 'write nullable presence')
                outcome(row.get('write'), 'write')
                if name == 'csharp.json':
                    outcome(row.get('construction'), 'construction')
                    require(type(row.get('indented')) is bool, 'indented boolean')
            elif op == 'read':
                require(type(row.get('input')) is str and type(row.get('returnedObject')) is bool, 'read fields')
                outcome(row.get('read'), 'read')
                outcome(row.get('rewrite'), 'rewrite')
                require((row['rewrite']['status'] != 'not-attempted') == row['returnedObject'], 'rewrite lifetime')
                if name == 'javascript.json':
                    for field, mode in [('runtimeFieldAccess', 'fixed-key'), ('originalKeyLookup', 'original-key')]:
                        v = row.get(field)
                        outcome(v, 'lookup', lookup=True)
                        require(v['queryMode'] == mode, 'lookup mode')
                        require(v['lookupKey'] is not None or (mode == 'original-key' and v['status'] == 'not-attempted'), 'lookup nullable key')
                        key = field + '/' + v['status']
                        lookups[key] = lookups.get(key, 0) + 1
            else:
                raise ValueError('unknown operation')
        require(json_equal(operations, inventory['operations']), 'operation inventory')
        require(json_equal(lookups, inventory['lookupOutcomes']), 'lookup inventory')
    for name, rows in index.items():
        if name == 'inputs.json':
            continue
        for row in rows.values():
            if row['operation'] != 'read':
                continue
            source = index.get(row['sourceCapture'], {}).get(row['sourceCaseId'])
            require(source is not None and source['kind'] == row['kind'], 'cross-runtime source')
            if row['sourceCapture'] == 'inputs.json':
                raw = source['input']
                require(row['origin'] == 'authored-stimulus', 'stimulus origin')
            else:
                require(source.get('operation') == 'write' and source['write']['status'] == 'accepted', 'linked successful write')
                raw = source['write']['value']
                if row['sourceCapture'] == 'csharp.json':
                    raw = '{"map":' + raw + '}'
                require(row['origin'] == ('actual-csharp-write' if row['sourceCapture'] == 'csharp.json' else 'actual-js-write'), 'write origin')
            require(row['input'] == raw, 'verbatim cross-runtime input')
    return data
