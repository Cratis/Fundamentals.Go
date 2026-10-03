# Copyright (c) Cratis. All rights reserved.
# Licensed under the MIT license. See LICENSE file in the project root for full license information.
"""Optional installed jsonschema check; normal Go CI needs neither Python nor runtimes."""
import copy
import json
import unittest
from validate import FIXTURES, load
from compare import contract

try:
    from jsonschema import Draft202012Validator
except ImportError:
    Draft202012Validator = None


@unittest.skipIf(Draft202012Validator is None, 'optional jsonschema not installed')
class Schema(unittest.TestCase):
    def test_schema_and_captures(self):
        schema = json.loads((FIXTURES / 'schema.json').read_text())
        Draft202012Validator.check_schema(schema)
        validator = Draft202012Validator(schema)
        for name, data in load(FIXTURES).items():
            if name == 'inputs.json':
                stimuli = dict(schema, **{'$ref': '#/$defs/stimuli'})
                for field in ['type', 'required', 'properties', 'additionalProperties', 'oneOf']:
                    stimuli.pop(field)
                Draft202012Validator(stimuli).validate(data)
            else:
                validator.validate(data)

    def test_js_reads_require_both_lookup_objects(self):
        validator = Draft202012Validator(json.loads((FIXTURES / 'schema.json').read_text()))
        data = load(FIXTURES)
        for name in ['csharp.json', 'csharp-cross.json', 'javascript.json']:
            self.assertTrue(validator.is_valid(data[name]))
        js = data['javascript.json']
        reads = [(i, r) for i, r in enumerate(js['observations']) if r['operation'] == 'read']
        self.assertEqual(len(reads), 167)
        self.assertEqual(sum(r['originalKeyLookup']['lookupKey'] is None for _, r in reads), 132)
        self.assertEqual({r['originalKeyLookup']['status'] for _, r in reads}, {'accepted', 'not-attempted'})
        for field in ['runtimeFieldAccess', 'originalKeyLookup']:
            for status in ['accepted', 'rejected', 'not-attempted']:
                candidates = [(i, r) for i, r in reads if r[field]['status'] == status]
                if field == 'originalKeyLookup' and status == 'rejected':
                    self.assertEqual(candidates, [])
                    continue
                self.assertTrue(candidates, (field, status))
                i, row = candidates[0]
                with self.subTest(field=field, status=status, id=row['id']):
                    damaged = copy.deepcopy(js)
                    del damaged['observations'][i][field]
                    self.assertFalse(validator.is_valid(damaged))
        # Authored stimuli have no independent key; failed linked reads still do.
        self.assertTrue(any(r['originalKeyLookup']['status'] == 'not-attempted' and
                            r['originalKeyLookup']['lookupKey'] is not None for _, r in reads))

    def test_schema_negative_capture(self):
        validator = Draft202012Validator(json.loads((FIXTURES / 'schema.json').read_text()))
        js = load(FIXTURES)['javascript.json']
        for change in [lambda d: d.update(count=True),
                       lambda d: d['observations'][0]['originalKeyLookup'].pop('lookupKey'),
                       lambda d: d['observations'][0]['originalKeyLookup'].pop('reason'),
                       lambda d: d['observations'][0]['rewrite'].update(stage='read')]:
            damaged = copy.deepcopy(js)
            change(damaged)
            self.assertFalse(validator.is_valid(damaged))


class Comparison(unittest.TestCase):
    def test_only_error_message_is_diagnostic(self):
        a = {'stage': 'read', 'status': 'rejected', 'exception': 'TypeError', 'innerException': None, 'message': 'first'}
        b = dict(a, message='second')
        self.assertEqual(contract(a), contract(b))
        self.assertNotEqual(contract(a), contract(dict(b, stage='write')))
        self.assertNotEqual(contract(a), contract(dict(b, exception='SyntaxError')))
        self.assertNotEqual(contract(a), contract({'stage': 'read', 'status': 'rejected', 'exception': 'TypeError', 'message': 'second'}))


if __name__ == '__main__':
    unittest.main(verbosity=2)
