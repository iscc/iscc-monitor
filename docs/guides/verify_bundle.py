# /// script
# requires-python = ">=3.10"
# dependencies = ["cryptography>=42"]
# ///
"""Verify an ISCC Monitor proof bundle without trusting the monitor.

Checks the hub's Ed25519 signature on the checkpoint, rebuilds the checkpoint's
Merkle root from the record and its RFC 6962 inclusion proof, and confirms that the
record names the bundle's ISCC-ID. When the bundle has an `anchor` member, it checks
that evidence the same way and writes the OpenTimestamps proof for `ots verify`.

    uv run verify_bundle.py MEIGKV5N6NTVF4AB.bundle.json
    uv run verify_bundle.py --resolve MEIGKV5N6NTVF4AB.bundle.json

With --resolve, the script also fetches the hub's did:web document and checks that it
lists the key that signed the checkpoint, so the key does not come from the monitor.
"""

import argparse
import base64
import hashlib
import json
import sys
import urllib.request
from pathlib import Path

from cryptography.exceptions import InvalidSignature
from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PublicKey

B58 = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"


def b58decode(text):
    """Decode a base58btc string to bytes."""
    num = 0
    for char in text:
        num = num * 58 + B58.index(char)
    raw = num.to_bytes((num.bit_length() + 7) // 8, "big")
    return b"\x00" * (len(text) - len(text.lstrip("1"))) + raw


def ed25519_key(multibase):
    """Return the 32-byte Ed25519 public key of a did:key style multibase string."""
    raw = b58decode(multibase.removeprefix("z"))
    if not multibase.startswith("z") or raw[:2] != b"\xed\x01" or len(raw) != 34:
        raise ValueError(f"not an Ed25519 multibase key: {multibase}")
    return raw[2:]


def key_hash(name, pubkey):
    """Return the signed-note key hash of an Ed25519 key: 4 bytes of SHA-256."""
    return hashlib.sha256(name.encode() + b"\n\x01" + pubkey).digest()[:4]


def open_checkpoint(note, pubkey):
    """Verify a signed checkpoint note and return its (origin, size, root, key id)."""
    text, sep, sigs = note.partition("\n\n")
    if not sep:
        raise ValueError("checkpoint has no signature block")
    text += "\n"
    for line in sigs.splitlines():
        name, _, sig_b64 = line.removeprefix("— ").rpartition(" ")
        sig = base64.b64decode(sig_b64)
        if sig[:4] != key_hash(name, pubkey):
            continue
        try:
            Ed25519PublicKey.from_public_bytes(pubkey).verify(sig[4:], text.encode())
        except InvalidSignature:
            raise ValueError("checkpoint signature does not verify") from None
        origin, size, root = text.splitlines()[:3]
        return origin, int(size), base64.b64decode(root), sig[:4].hex()
    raise ValueError("no signature by the bundle's key")


def root_from_proof(leaf, index, size, proof):
    """Rebuild a Merkle tree root from a leaf hash and its inclusion proof (RFC 9162, 2.1.3.2)."""
    if index >= size:
        raise ValueError(f"leaf index {index} is outside a tree of size {size}")
    fn, sn, node = index, size - 1, leaf
    for sibling in proof:
        if sn == 0:
            raise ValueError("inclusion proof is too long")
        if fn % 2 == 1 or fn == sn:
            node = hashlib.sha256(b"\x01" + sibling + node).digest()
            while fn % 2 == 0 and fn != 0:
                fn, sn = fn >> 1, sn >> 1
        else:
            node = hashlib.sha256(b"\x01" + node + sibling).digest()
        fn, sn = fn >> 1, sn >> 1
    if sn != 0:
        raise ValueError("inclusion proof is too short")
    return node


def check_evidence(label, checkpoint, inclusion, key, record):
    """Check one checkpoint, its inclusion proof and the record; return the checkpoint root."""
    origin, size, root, key_id = open_checkpoint(checkpoint, ed25519_key(key["multibase"]))
    if inclusion["treeSize"] != size:
        raise ValueError(f"{label}: proof is for size {inclusion['treeSize']}, checkpoint has {size}")
    leaf = hashlib.sha256(b"\x00" + record).digest()
    proof = [base64.b64decode(h) for h in inclusion["inclusionProof"]]
    if root_from_proof(leaf, inclusion["leafIndex"], size, proof) != root:
        raise ValueError(f"{label}: inclusion proof does not rebuild the checkpoint root")
    print(f"ok  {label}: signed by {origin} key {key_id}, record at {inclusion['leafIndex']} of {size}")
    return root


def resolve_keys(domain):
    """Return the Ed25519 multibase keys listed in a hub's did:web document."""
    with urllib.request.urlopen(f"https://{domain}/.well-known/did.json", timeout=10) as resp:  # noqa: S310
        doc = json.load(resp)
    return {vm.get("publicKeyMultibase") for vm in doc.get("verificationMethod", [])}


def main():
    """Verify the bundle named on the command line and report each check."""
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("bundle", type=Path)
    parser.add_argument("--resolve", action="store_true", help="check the key against the hub's did.json")
    args = parser.parse_args()
    bundle = json.loads(args.bundle.read_text(encoding="utf-8"))

    record = base64.b64decode(bundle["record"])
    if json.loads(record)["iscc_id"] != bundle["iscc_id"]:
        raise ValueError("the record does not name the bundle's ISCC-ID")
    print(f"ok  record names {bundle['iscc_id']}")
    check_evidence("checkpoint", bundle["checkpoint"], bundle["inclusion"], bundle["key"], record)

    anchor = bundle.get("anchor")
    if anchor:
        root = check_evidence("anchor", anchor["checkpoint"], anchor["inclusion"], anchor["key"], record)
        ots = args.bundle.with_name(args.bundle.name.split(".")[0] + ".anchor.ots")
        ots.write_bytes(base64.b64decode(anchor["ots"]))
        print(f"    check the Bitcoin anchor: ots verify -d {root.hex()} {ots.name}")

    if args.resolve:
        keys = {bundle["key"]["multibase"]} | ({anchor["key"]["multibase"]} if anchor else set())
        missing = keys - resolve_keys(bundle["hub"]["domain"])
        if missing:
            raise ValueError(f"did:web:{bundle['hub']['domain']} does not list {', '.join(sorted(missing))}")
        print(f"ok  did:web:{bundle['hub']['domain']} lists the signing key")


if __name__ == "__main__":
    try:
        main()
    except (ValueError, KeyError) as exc:
        sys.exit(f"FAIL {exc}")
