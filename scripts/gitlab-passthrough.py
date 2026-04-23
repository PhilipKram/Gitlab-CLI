#!/usr/bin/env python3
"""
TCP passthrough from Mac:443 -> <UPSTREAM_HOST>:443.

Lets Docker containers (without VPN access) reach a private GitLab via the Mac.
No TLS termination — the real handshake happens end-to-end, so certificate
validation still works against the real upstream cert.

Configuration:
  UPSTREAM_HOST   required. Hostname of the upstream GitLab (e.g. gitlab.example.com).
                  Can also be passed as argv[1].
  UPSTREAM_PORT   optional. Defaults to 443.
  LISTEN_PORT     optional. Defaults to 443.

Run:  sudo UPSTREAM_HOST=gitlab.example.com python3 gitlab-passthrough.py
"""

import os
import socket
import sys
import threading


def load_config() -> tuple[tuple[str, int], tuple[str, int]]:
    host = os.environ.get("UPSTREAM_HOST") or (sys.argv[1] if len(sys.argv) > 1 else "")
    if not host:
        sys.exit("UPSTREAM_HOST must be set (env var or argv[1])")
    upstream_port = int(os.environ.get("UPSTREAM_PORT", "443"))
    listen_port = int(os.environ.get("LISTEN_PORT", "443"))
    return ("0.0.0.0", listen_port), (host, upstream_port)


def pipe(src: socket.socket, dst: socket.socket) -> None:
    try:
        while True:
            data = src.recv(8192)
            if not data:
                break
            dst.sendall(data)
    except OSError:
        pass
    finally:
        for s in (src, dst):
            try:
                s.shutdown(socket.SHUT_RDWR)
            except OSError:
                pass
            try:
                s.close()
            except OSError:
                pass


def handle(client: socket.socket, addr: tuple, upstream_addr: tuple[str, int]) -> None:
    try:
        upstream = socket.create_connection(upstream_addr, timeout=10)
    except OSError as e:
        print(f"[{addr[0]}:{addr[1]}] upstream connect failed: {e}", file=sys.stderr)
        client.close()
        return
    print(f"[{addr[0]}:{addr[1]}] -> {upstream_addr[0]}:{upstream_addr[1]}")
    threading.Thread(target=pipe, args=(client, upstream), daemon=True).start()
    threading.Thread(target=pipe, args=(upstream, client), daemon=True).start()


def main() -> None:
    listen, upstream = load_config()
    server = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    server.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    server.bind(listen)
    server.listen(16)
    print(f"listening on {listen[0]}:{listen[1]} -> {upstream[0]}:{upstream[1]}", flush=True)
    try:
        while True:
            client, addr = server.accept()
            threading.Thread(target=handle, args=(client, addr, upstream), daemon=True).start()
    except KeyboardInterrupt:
        print("\nshutting down", file=sys.stderr)
    finally:
        server.close()


if __name__ == "__main__":
    main()
