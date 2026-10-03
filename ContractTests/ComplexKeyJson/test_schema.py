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
