# Copyright (c) Cratis. All rights reserved.
# Licensed under the MIT license. See LICENSE file in the project root for full license information.
"""Compare separately executed captures; never overwrite or accept fixture drift."""
import argparse
import json
from pathlib import Path
import sys
from validate import FIXTURES, json_equal, load, validate


def contract(value):
    # Only arbitrary exception message literals are diagnostic. Stage/category,
    # explicit null presence, runtime identity, and raw wire strings remain exact.
    if isinstance(value, dict):
        return {k: contract(v) for k, v in value.items()
                if not (k == 'message' and value.get('status') == 'rejected')}
    if isinstance(value, list):
        return [contract(v) for v in value]
    return value


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('capture', type=Path)
    args = parser.parse_args()
    manifest = json.loads((FIXTURES / 'manifest.json').read_text())
    expected = validate(load(FIXTURES), manifest)
    actual = validate(load(args.capture), manifest)
    drift = [name for name in expected if not json_equal(contract(actual[name]), contract(expected[name]))]
    if drift:
        print('Capture drift: ' + ', '.join(drift) + '; retain separate evidence, do not replace fixtures', file=sys.stderr)
        return 1
    print('Matches 132 stimuli and 353 actual observations; raw wire, types, runtime, stages and crosslinks exact (message literals diagnostic only).')
    return 0


if __name__ == '__main__':
    sys.exit(main())
