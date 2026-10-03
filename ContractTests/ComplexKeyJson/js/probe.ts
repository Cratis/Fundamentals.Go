// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.
import { JsonSerializer } from './source/JsonSerializer';
import { ValueMap } from './source/ValueMap';
import { ConceptAs } from './source/ConceptAs';
import { Guid } from './source/Guid';
import { field } from './source/fieldDecorator';
import { Fields } from './source/Fields';

declare function require(name: string): any;
declare const process: any;
const fs = require('node:fs');
const path = require('node:path');
const root = process.argv[2];
const compiler = process.argv[3] || process.env.TSC;
if (!compiler) throw new Error('Supply TSC compiler path as argument or environment');
const typescript = require('node:child_process').execFileSync(process.execPath, [compiler, '--version'], { encoding: 'utf8' }).trim().replace(/^Version /, '');

class TextKey extends ConceptAs<string> { }
class IdKey extends ConceptAs<Guid> { static readonly valueType = Guid; }
class Composite {
    @field(Guid) id!: Guid;
    @field(String) name!: string;
}
class Payload {
    @field(Number) number!: number;
    @field(String) label!: string;
}
class StringEnvelope {
    @field(ValueMap, { genericArguments: [String, Number] }) map = new ValueMap<string, number>();
}
class TextEnvelope {
    @field(ValueMap, { genericArguments: [TextKey, Number] }) map = new ValueMap<TextKey, number>();
}
class GuidEnvelope {
    @field(ValueMap, { genericArguments: [Guid, Number] }) map = new ValueMap<Guid, number>();
}
class IdEnvelope {
    @field(ValueMap, { genericArguments: [IdKey, Number] }) map = new ValueMap<IdKey, number>();
}
class CompositeEnvelope {
    @field(ValueMap, { genericArguments: [Composite, Number] }) map = new ValueMap<Composite, number>();
}
class ObjectValueEnvelope {
    @field(ValueMap, { genericArguments: [Composite, Payload] }) map = new ValueMap<Composite, Payload>();
}
const kinds: Record<string, any> = { string: StringEnvelope, stringConcept: TextEnvelope, guid: GuidEnvelope, guidConcept: IdEnvelope, composite: CompositeEnvelope, compositeObjectValue: ObjectValueEnvelope };
const observations: any[] = [];
const observe = (stage: string, action: () => any) => {
    try { return { stage, status: 'accepted', value: action() }; }
    catch (error: any) { return { stage, status: 'rejected', exception: error.constructor.name, innerException: null, message: error.message }; }
};
const notAttempted = (stage: string, reason: string) => ({ stage, status: 'not-attempted', reason });
const describe = (value: any): any => {
    if (value === undefined) return { state: 'undefined' };
    if (value === null) return { state: 'null' };
    if (value instanceof ConceptAs) return { type: value.constructor.name, value: describe(value.value) };
    if (value instanceof Guid) return { type: 'Guid', value: value.toString() };
    if (value instanceof Composite) return { type: 'Composite', id: describe(value.id), name: describe(value.name), ownPropertyOrder: Object.keys(value), nativeStringify: JSON.stringify(value) };
    if (value instanceof Payload) return { type: 'Payload', number: describe(value.number), label: describe(value.label) };
    return { type: typeof value, value };
};
const describeMap = (map: any): any => {
    if (map === undefined || map === null) return describe(map);
    const entries = [...map.entries()];
    return { state: 'map', runtimeType: map.constructor.name, count: entries.length, entries: entries.map(([key, value]) => ({ key: describe(key), value: describe(value) })) };
};
function read(id: string, kind: string, raw: string, origin: string, sourceCapture: string, sourceCaseId: string, originalKey?: any) {
    let returned: any;
    const result = observe('read', () => {
        returned = JsonSerializer.deserialize(kinds[kind], raw);
        return describeMap(returned.map);
    });
    // Historical fixed-key probes are retained, not reinterpreted as original-key queries.
    const fixedKey = expectedKey(kind);
    const access = { queryMode: 'fixed-key', lookupKey: describe(fixedKey), ...(returned === undefined ? notAttempted('lookup', 'no-returned-object') : observe('lookup', () => describe(returned.map.get(fixedKey)))) };
    const originalKeyLookup = { queryMode: 'original-key', lookupKey: originalKey === undefined ? null : describe(originalKey), ...(originalKey === undefined ? notAttempted('lookup', 'no-independent-write-key') : returned === undefined ? notAttempted('lookup', 'no-returned-object') : observe('lookup', () => describe(returned.map.get(originalKey)))) };
    const rewrite = returned === undefined ? notAttempted('rewrite', 'no-returned-object') : observe('rewrite', () => JsonSerializer.serialize(returned));
    observations.push({ id, kind, operation: 'read', origin, sourceCapture, sourceCaseId, input: raw, read: result, runtimeFieldAccess: access, originalKeyLookup, rewrite, returnedObject: returned !== undefined });
}
const guidText = 'e094c582-9d91-4df0-b795-8d39fce089da';
const composite = (name: string | null = 'tenant-a') => Object.assign(new Composite(), { id: Guid.parse(guidText), name });
function expectedKey(kind: string): any {
    if (kind === 'string') return 'plain';
    if (kind === 'stringConcept') return new TextKey('plain');
    if (kind === 'guid') return Guid.parse(guidText);
    if (kind === 'guidConcept') return new IdKey(Guid.parse(guidText));
    return composite();
}
function write(id: string, kind: string, key: any, value: any) {
    const envelope = new kinds[kind]();
    envelope.map.set(key, value);
    const write = observe('write', () => JsonSerializer.serialize(envelope));
    observations.push({ id, kind, operation: 'write', sourceCapture: null, sourceCaseId: null, inputKey: describe(key), inputValue: describe(value), write });
    if (write.status === 'accepted') read('self/' + id, kind, write.value, 'actual-js-write', 'javascript.json', id, key);
}
const cases = JSON.parse(fs.readFileSync(path.join(root, 'inputs.json'), 'utf8'));
for (const row of cases) read(row.id, row.kind, row.input, 'authored-stimulus', 'inputs.json', row.id);
function csharpOriginalKey(kind: string, key: any): any {
    if (kind === 'string') return key;
    if (kind === 'stringConcept') return new TextKey(key.value);
    if (kind === 'guid') return Guid.parse(key.value);
    if (kind === 'guidConcept') return new IdKey(Guid.parse(key.value));
    return Object.assign(new Composite(), { id: Guid.parse(key.id), name: key.name });
}
const cs = JSON.parse(fs.readFileSync(path.join(root, 'csharp.json'), 'utf8'));
for (const row of cs.observations) {
    if (row.operation === 'write' && row.write.status === 'accepted')
        read('cs/' + row.id, row.kind, '{"map":' + row.write.value + '}', 'actual-csharp-write', 'csharp.json', row.id, csharpOriginalKey(row.kind, row.inputKey));
}
for (const [index, text] of ['plain', 'a"b', 'a\\b', 'é雪😀<>&\'"', ''].entries()) {
    write('string/write/' + index, 'string', text, 42);
    write('stringConcept/write/' + index, 'stringConcept', new TextKey(text), 42);
}
write('guid/write', 'guid', Guid.parse(guidText), 42);
write('guidConcept/write', 'guidConcept', new IdKey(Guid.parse(guidText)), 42);
write('guidConcept/write-zero', 'guidConcept', new IdKey(Guid.empty), 0);
write('composite/write', 'composite', composite(), 42);
write('composite/write-escaped', 'composite', composite('a"\\é雪😀<>&'), 42);
write('composite/write-null-member', 'composite', composite(null), 42);
write('compositeObjectValue/write', 'compositeObjectValue', composite(), Object.assign(new Payload(), { number: 42, label: 'answer' }));
write('compositeObjectValue/write-null', 'compositeObjectValue', composite(), null);

// Plain-object key equality is intentionally a different witness from metadata-normalized typed reads.
const a = { id: guidText, name: 'tenant-a' };
const b = { name: 'tenant-a', id: guidText };
const map = new ValueMap<any, number>().set(a, 1).set(b, 2);
observations.push({ id: 'equality/plain-object-property-order', operation: 'equality', sourceCapture: null, sourceCaseId: null, first: JSON.stringify(a), second: JSON.stringify(b), count: [...map.entries()].length, lookupFirst: map.get(a), lookupFreshFirst: map.get({ id: guidText, name: 'tenant-a' }), lookupSecond: map.get(b) });

// Direct factory-free reads would exercise the empty fallback, not typed-field parity.
const metadata = Object.entries(kinds).map(([kind, type]) => {
    const fields = Fields.getFieldsForType(type);
    if (fields.length !== 1 || fields[0].type !== ValueMap || fields[0].genericArguments.length !== 2) throw new Error('Missing real typed field metadata: ' + kind);
    return { kind, field: fields[0].name, type: fields[0].type.name, genericArguments: fields[0].genericArguments.map(t => t.name) };
});
if (!observations.length || new Set(observations.map(r => r.id)).size !== observations.length) throw new Error('Empty/duplicate observations');
fs.writeFileSync(path.join(root, 'javascript.json'), JSON.stringify({ schemaVersion: 1, sourceRevision: 'd2accc4a79b6bcf2708213c97093ab5ba6c06381', node: process.version, typescript, profile: 'Exact pinned TS import closure; legacy decorators; own emitted fields; typed ValueMap; no npm package equivalence claim', count: observations.length, metadata, observations }, null, 2) + '\n');
console.log(`Captured ${observations.length} unique JS observations with ${metadata.length} verified typed fields`);
