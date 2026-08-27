#!/usr/bin/env python3
"""Scrub a stripeek capture into a fixture safe to hand to strangers.

    python3 tools/stripeek_fixture.py [path/to/demo-history.json]

Stripe object IDs embed a fragment derived from the account that created them,
so a capture leaks the account id in every single id it contains. The traffic
itself is test-mode and stripeek already redacts Authorization headers, so the
only thing to remove is that identity.

Substitution is global and consistent, and ids are opaque strings, so every
cross-reference survives — "related calls" and the group links still work.
"""
import base64
import json
import sys

SOURCE = sys.argv[1] if len(sys.argv) > 1 else "../stripeek/demo/demo-history.json"
TARGET = "visitor/stripeek-history.json"

# Replace the account id first: it contains the fragment.
SWAPS = [
    ("acct_1AGUCWB3ZHLBhbGB", "acct_1D3MOxDem0AccT99"),
    ("B3ZHLBhbGB", "Dem0AccT99"),
]
BODY_FIELDS = ("ReqBody", "RespBody")


def scrub(text):
    for old, new in SWAPS:
        text = text.replace(old, new)
    return text


def scrub_entry(entry):
    out = {}
    for key, value in entry.items():
        if key in BODY_FIELDS and isinstance(value, str) and value:
            decoded = base64.b64decode(value).decode("utf-8", "surrogateescape")
            out[key] = base64.b64encode(
                scrub(decoded).encode("utf-8", "surrogateescape")
            ).decode("ascii")
        elif isinstance(value, str):
            out[key] = scrub(value)
        else:
            out[key] = json.loads(scrub(json.dumps(value)))
    return out


calls = [scrub_entry(e) for e in json.load(open(SOURCE))]
with open(TARGET, "w") as f:
    json.dump(calls, f)

# Prove it: nothing of the original identity may survive, anywhere.
raw = open(TARGET).read()
haystack = [raw]
for entry in calls:
    for field in BODY_FIELDS:
        if entry.get(field):
            haystack.append(base64.b64decode(entry[field]).decode("utf-8", "replace"))
blob = "\n".join(haystack)
for old, _ in SWAPS:
    assert old not in blob, f"{old} survived the scrub"
print(f"{TARGET}: {len(calls)} calls, {len(raw) / 1024:.0f} KB, account identity removed")
