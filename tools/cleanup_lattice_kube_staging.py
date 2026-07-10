#!/usr/bin/env python3
"""Clean up LatticeKube staging resources by prefix.

Dry-run by default. Pass --execute to mutate staging.
"""

from __future__ import annotations

import argparse
import json
import os
import ssl
import sys
import time
import urllib.error
import urllib.parse
import urllib.request


def env(name: str, default: str = "") -> str:
    return os.environ.get(name, default).strip()


class LatticeClient:
    def __init__(self, endpoint: str, api_key: str, insecure: bool):
        self.endpoint = endpoint.rstrip("/")
        self.api_key = api_key
        self.context = ssl._create_unverified_context() if insecure else None

    def request(self, method: str, path: str, body: dict | None = None):
        data = None if body is None else json.dumps(body).encode()
        req = urllib.request.Request(self.endpoint + path, data=data, method=method)
        req.add_header("Content-Type", "application/json")
        if self.api_key:
            req.add_header("Authorization", "Bearer " + self.api_key)
        try:
            with urllib.request.urlopen(req, context=self.context, timeout=60) as resp:
                raw = resp.read()
                if not raw:
                    return None
                return json.loads(raw.decode())
        except urllib.error.HTTPError as exc:
            raw = exc.read().decode(errors="replace")
            raise RuntimeError(f"{method} {path} failed: HTTP {exc.code}: {raw}") from exc

    def get(self, path: str):
        return self.request("GET", path)

    def delete(self, path: str):
        return self.request("DELETE", path)


def starts_with_prefix(value, prefix: str) -> bool:
    return isinstance(value, str) and value.startswith(prefix)


def describe(resource: dict) -> str:
    bits = [resource.get("id", "")]
    if resource.get("name"):
        bits.append(resource["name"])
    if resource.get("ip"):
        bits.append(resource["ip"])
    if resource.get("description"):
        bits.append(resource["description"])
    return " ".join(str(b) for b in bits if b)


def delete_resources(client: LatticeClient, label: str, resources: list[dict], path_fn, execute: bool):
    if not resources:
        print(f"{label}: none")
        return
    for resource in resources:
        print(f"{label}: {'delete' if execute else 'would delete'} {describe(resource)}")
        if execute:
            try:
                client.delete(path_fn(resource))
            except RuntimeError as exc:
                print(f"  warning: {exc}", file=sys.stderr)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--prefix", required=True, help="resource name/description prefix to delete")
    parser.add_argument("--execute", action="store_true", help="actually delete resources")
    parser.add_argument("--endpoint", default=env("LATTICE_ENDPOINT"), help="defaults to LATTICE_ENDPOINT")
    parser.add_argument("--api-key", default=env("LATTICE_API_KEY"), help="defaults to LATTICE_API_KEY")
    parser.add_argument("--insecure", action="store_true", default=env("LATTICE_INSECURE").lower() == "true")
    args = parser.parse_args()

    if not args.endpoint:
        print("missing --endpoint or LATTICE_ENDPOINT", file=sys.stderr)
        return 2

    client = LatticeClient(args.endpoint, args.api_key, args.insecure)
    prefix = args.prefix
    print(f"mode: {'execute' if args.execute else 'dry-run'}")
    print(f"prefix: {prefix}")

    clusters = [
        c for c in client.get("/kube/clusters")
        if starts_with_prefix(c.get("name"), prefix)
    ]
    delete_resources(client, "clusters", clusters, lambda c: f"/kube/clusters/{c['id']}", args.execute)

    if args.execute and clusters:
        time.sleep(3)

    vms = [
        vm for vm in client.get("/vm")
        if starts_with_prefix(vm.get("name"), prefix)
    ]
    delete_resources(client, "vms", vms, lambda vm: f"/vm/{vm['id']}?force=true", args.execute)

    public_ips = [
        ip for ip in client.get("/network/public-ips")
        if starts_with_prefix(ip.get("description"), prefix)
    ]
    delete_resources(client, "public_ips", public_ips, lambda ip: f"/network/public-ips/{ip['id']}", args.execute)

    pools = [
        p for p in client.get("/network/public-ip-pools")
        if starts_with_prefix(p.get("name"), prefix)
    ]
    delete_resources(client, "public_ip_pools", pools, lambda p: f"/network/public-ip-pools/{p['id']}", args.execute)

    vpcs = [
        vpc for vpc in client.get("/vpc")
        if starts_with_prefix(vpc.get("name"), prefix)
    ]
    delete_resources(client, "vpcs", vpcs, lambda vpc: f"/vpc/{vpc['id']}", args.execute)

    if not args.execute:
        print("dry-run only; pass --execute to delete")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
