# Copyright (c) Cratis. All rights reserved.
# Licensed under the MIT license. See LICENSE file in the project root for full license information.
"""Assert selected contract observations and real cross-runtime input links, not codec parity."""
import json
from pathlib import Path
import unittest
import copy
import os
from validate import FIXTURES, json_equal, load, validate

ROOT = Path(os.environ.get('CAPTURE_ROOT', FIXTURES))

class Captures(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.cs = json.loads((ROOT / 'csharp.json').read_text())
        cls.js = json.loads((ROOT / 'javascript.json').read_text())
        cls.cross = json.loads((ROOT / 'csharp-cross.json').read_text())
        cls.c = {r['id']: r for r in cls.cs['observations']}
        cls.j = {r['id']: r for r in cls.js['observations']}
        cls.x = {r['id']: r for r in cls.cross['observations']}

    def test_dataset_invariants(self):
        validate(load(ROOT), json.loads((FIXTURES / 'manifest.json').read_text()))

    def test_negative_fixtures(self):
        data = load(ROOT)
        manifest = json.loads((FIXTURES / 'manifest.json').read_text())
        mutations = [
            ('empty', lambda d: d['csharp.json'].update(observations=[])),
            ('truncated', lambda d: d['javascript.json']['observations'].pop()),
            ('duplicate', lambda d: d['csharp.json']['observations'].__setitem__(1, d['csharp.json']['observations'][0])),
            ('bool-count', lambda d: d['csharp-cross.json'].update(count=True)),
            ('missing-nullable', lambda d: d['javascript.json']['observations'][0]['originalKeyLookup'].pop('lookupKey')),
            ('missing-link', lambda d: d['javascript.json']['observations'][0].pop('sourceCaseId')),
            ('wrong-stage', lambda d: d['javascript.json']['observations'][0]['rewrite'].update(stage='read')),
            ('missing-reason', lambda d: d['javascript.json']['observations'][0]['originalKeyLookup'].pop('reason')),
            ('cross-wire', lambda d: d['csharp-cross.json']['observations'][0].update(input='{}')),
            ('wrong-type-metadata', lambda d: d['javascript.json']['metadata'][0]['genericArguments'].__setitem__(0, 'Guid')),
        ]
        for name, mutate in mutations:
            with self.subTest(name=name):
                damaged = copy.deepcopy(data)
                mutate(damaged)
                with self.assertRaises((ValueError, KeyError)):
                    validate(damaged, manifest)

    def test_comparison_is_type_strict(self):
        for actual, expected in [(True, 1), (False, 0), (1.0, 1), ({'x': True}, {'x': 1}), ([True], [1]), ({}, {'x': None})]:
            self.assertFalse(json_equal(actual, expected))
        self.assertTrue(json_equal({'x': [1, None, True]}, {'x': [1, None, True]}))

    def test_original_key_lookups_do_not_reinterpret_fixed_probes(self):
        for ident, expected in [('self/guidConcept/write-zero', 0), ('self/composite/write-escaped', 42), ('self/stringConcept/write/1', 42)]:
            row = self.j[ident]
            self.assertEqual(row['runtimeFieldAccess']['queryMode'], 'fixed-key')
            self.assertEqual(row['runtimeFieldAccess']['value'], {'state': 'undefined'})
            self.assertEqual(row['originalKeyLookup']['queryMode'], 'original-key')
            self.assertEqual(row['originalKeyLookup']['value'], {'type': 'number', 'value': expected})
        row = self.j['self/guid/write']
        self.assertEqual(row['originalKeyLookup']['lookupKey'], row['runtimeFieldAccess']['lookupKey'])
        self.assertEqual(row['originalKeyLookup']['value'], {'state': 'undefined'})
        self.assertEqual(row['sourceCaseId'], 'guid/write')

    def test_nonempty_counts_ids_and_actual_metadata(self):
        for data, count in [(self.cs, 151), (self.js, 186), (self.cross, 16)]:
            self.assertEqual(data['count'], count)
            self.assertEqual(len(data['observations']), count)
            self.assertEqual(len({r['id'] for r in data['observations']}), count)
        self.assertEqual(self.cs['globalStubAccesses'], 0)
        self.assertEqual(self.cross['globalStubAccesses'], 0)
        self.assertEqual(len(self.js['metadata']), 6)
        for field in self.js['metadata']:
            self.assertEqual(field['type'], 'ValueMap')
            self.assertEqual(len(field['genericArguments']), 2)

    def test_cross_reads_use_the_actual_writes_verbatim(self):
        writes = [r for r in self.cs['observations'] if r['operation'] == 'write']
        self.assertEqual(len(writes), 19)
        for row in writes:
            self.assertEqual(self.j['cs/' + row['id']]['input'], '{"map":' + row['write']['value'] + '}')
        writes = [r for r in self.js['observations'] if r['operation'] == 'write' and r['write']['status'] == 'accepted']
        self.assertEqual(len(writes), 16)
        for row in writes:
            self.assertEqual(self.x['js/' + row['id']]['input'], row['write']['value'])

    def test_factory_admits_concept_but_not_string_guid_or_readonly_interface(self):
        rows = self.cs['factoryAdmission']
        self.assertEqual(len(rows), 11)
        self.assertEqual([r['admitted'] for r in rows], [False, False, False, False, False, False, True, True, False, True, False])

    def test_concept_string_has_embedded_json_while_string_does_not(self):
        plain = self.c['string/write/132']['write']['value']
        concept = self.c['stringConcept/write/132']['write']['value']
        self.assertEqual(list(json.loads(plain)), ['plain'])
        self.assertEqual(list(json.loads(concept)), ['"plain"'])
        self.assertEqual(self.j['cs/stringConcept/write/132']['runtimeFieldAccess']['value']['value'], 42)

    def test_duplicate_decoded_keys_and_typed_property_order_are_last_wins(self):
        for ident in ['stringConcept/duplicate-decoded-escape', 'composite/reordered-duplicate', 'composite/whitespace-duplicate', 'composite/duplicate-json-property']:
            for data in [self.c, self.j]:
                value = data[ident]['read']['value']
                self.assertEqual(value['count'], 1)
                entry_value = value['entries'][0]['value']
                self.assertEqual(entry_value if isinstance(entry_value, int) else entry_value['value'], 2)
        self.assertEqual(self.j['equality/plain-object-property-order']['count'], 2)

    def test_missing_and_null_fields_do_not_become_js_empty_maps(self):
        self.assertEqual(self.c['composite/missing']['read']['value']['state'], 'map')
        self.assertEqual(self.j['composite/missing']['read']['value']['state'], 'undefined')
        self.assertEqual(self.j['composite/missing']['runtimeFieldAccess']['status'], 'rejected')
        self.assertEqual(self.j['composite/null-map']['read']['value']['state'], 'null')
        self.assertEqual(self.j['composite/null-map']['rewrite']['status'], 'rejected')

    def test_invalid_shapes_and_number_values_are_not_strict_in_js(self):
        self.assertEqual(self.c['composite/number-map']['read']['status'], 'rejected')
        self.assertEqual(self.j['composite/number-map']['read']['value']['count'], 0)
        self.assertEqual(self.c['composite/invalid-value']['read']['status'], 'rejected')
        self.assertEqual(self.j['composite/invalid-value']['read']['value']['entries'][0]['value']['type'], 'string')

    def test_bare_guid_js_write_is_not_a_round_trip(self):
        self.assertEqual(self.j['cs/guid/write']['runtimeFieldAccess']['value']['value'], 42)
        self.assertEqual(self.j['self/guid/write']['read']['status'], 'accepted')
        self.assertEqual(self.j['self/guid/write']['runtimeFieldAccess']['value']['state'], 'undefined')
        self.assertEqual(self.j['self/guid/write']['read']['value']['entries'][0]['key']['value'], '094c5802-91NaN-f09d-4db7-958d39fce089')
        self.assertEqual(self.x['js/guid/write']['read']['status'], 'rejected')
        self.assertEqual(self.x['js/guidConcept/write']['read']['status'], 'accepted')

    def test_invalid_later_key_does_not_return_partial_object(self):
        for data in [self.c, self.j]:
            for kind in ['stringConcept', 'guidConcept', 'composite']:
                row = data[kind + '/good-then-malformed-key']
                self.assertEqual(row['read']['status'], 'rejected')
                self.assertFalse(row['returnedObject'])

    def test_null_object_value_and_nullable_member_are_not_bidirectionally_safe(self):
        self.assertEqual(self.c['compositeObjectValue/write-null']['write']['status'], 'accepted')
        self.assertEqual(self.j['cs/compositeObjectValue/write-null']['read']['status'], 'rejected')
        self.assertEqual(self.j['cs/composite/write-null-member']['read']['status'], 'accepted')
        self.assertEqual(self.j['cs/composite/write-null-member']['rewrite']['status'], 'rejected')

if __name__ == '__main__':
    unittest.main(verbosity=2)
