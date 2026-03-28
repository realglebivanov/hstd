# xrayvpn

Split-routing VPN for Linux using [Xray-core](https://github.com/XTLS/Xray-core) with VLESS+REALITY. Routes traffic for Russian IP ranges directly and tunnels everything else. Geodata and CIDR lists refresh daily.

## Architecture

Three hosts work together:

```
┌─────────────────────────────────────────────────────────────┐
│ hstd  (router/client)                                       │
│                                                             │
│  process traffic                                            │
│       │                                                     │
│  ip rules ──── service users ──────────────► direct table  │
│  (transmission, navidrome)                  (bypass tunnel) │
│       │                                                     │
│       │  everything else                                    │
│       ▼                                                     │
│  default route → xray0 TUN                                  │
│       │                                                     │
│  tun2socksd        reads TUN packets,                       │
│       │            forwards to SOCKS5                       │
│       │  SOCKS5 :1080                                       │
│       ▼                                                     │
│  xrayvpnd          embeds xray-core                         │
│   ├── RU IPs ──────────────────────────────► direct table  │
│   │                (marked xray_out_mark,    (real gateway) │
│   │                 ip rule bypasses TUN)                   │
│   └── other IPs                                             │
└───────────────────────────┬─────────────────────────────────┘
                            │ VLESS+REALITY :443
                            ▼
┌─────────────────────────────────────────────────────────────┐
│ xray_server                                                 │
│                                                             │
│  xray-core                                                  │
│    VLESS+REALITY inbound                                    │
│    REALITY impersonates a real TLS destination              │
│                                                             │
│  clientrotate.timer                                         │
│    rotates all client UUIDs daily                           │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│ xray_proxy                                                  │
│                                                             │
│  nginx                                                      │
│    :443 stream ──────────────────────► xray_server:443      │
│    :80  HTTP (certbot ACME only)                            │
│                                                             │
│  subsrv  :8080 (HTTPS, Let's Encrypt)                       │
│    GET /<encoded-link>  → VLESS config JSON                 │
│    GET /admin/          → admin panel                       │
└──────────────────┬──────────────────────────────────────────┘
                   │
         ┌─────────┴─────────┐
         │  mobile clients   │
         │                   │
         │  1. GET /link     │
         │     ← VLESS JSON  │
         │                   │
         │  2. VLESS+REALITY │
         │     → xray_proxy  │
         │       (nginx :443)│
         │       → xray_server
         └───────────────────┘
```

### hstd routing in detail

`tun2socksd` sets up three layers when the tunnel starts and tears them down on stop:

**Route tables**

- Main table default route replaced with `default via xray0` — all traffic enters the TUN
- Separate `direct` route table retains the original gateway for bypass traffic

**IP rules** (evaluated top-down)

| Priority | Match | Table | Effect |
|---|---|---|---|
| 1000+ | service user UID + dst in APD CIDR | main | local network access for services |
| 2000+ | service user UID | direct | service internet traffic bypasses tunnel |
| 3000 | mark == `xray_out_mark` | direct | xray-core's own outbound bypasses tunnel (loop prevention) |

**nftables `xray_vpn` table** (added dynamically by tun2socksd)

Marks forwarded traffic through the TUN with `xray_traffic_mark` and accepts it. Allows forwarding on paths: `lo→xray0`, `apd→xray0`, `xray0→wan`, `xray0→apd`. The static nftables config admits packets carrying `xray_traffic_mark` in the forward chain.

**Packet walk — non-Russian traffic**

```
process → kernel → default route → xray0 TUN
  → tun2socksd reads packet → SOCKS5 :1080
  → xrayvpnd (xray-core) → VLESS outbound, marked xray_out_mark
  → ip rule: xray_out_mark → direct table → real gateway → internet
```

**Russian IPs** — xray-core routes to a direct outbound also marked `xray_out_mark`, exits via real gateway, never touches VLESS.

**Service users** (transmission, navidrome) — bypassed at the ip rule level before reaching the TUN.

## Deploy

Deployment uses [pyinfra](https://pyinfra.com) from the repo root. Secrets are read from the [`pass`](https://www.passwordstore.org/) password store.

```sh
# deploy all hosts
pyinfra inventories/all.py deploy.py

# deploy a single role
pyinfra inventories/hstd.py deploy.py
pyinfra inventories/xray_server.py deploy.py
pyinfra inventories/xray_proxy.py deploy.py
```

### Secrets (via `pass`)

| Entry | Used for |
|---|---|
| `rotate_secret` | HMAC root secret for client UUID derivation |
| `wpa_passphrase` | WiFi passphrase (hstd) |
| `sub_path` | Legacy subscription URL path |
| `reality_private_key` | REALITY private key (xray_server) |
| `admin_user` | subsrv admin username |
| `admin_password_hash` | bcrypt hash of subsrv admin password |

Generate a password hash:

```sh
htpasswd -bnBC 10 "" yourpassword | tr -d ':\n'
```

### hstd inventory vars

| Var | Description |
|---|---|
| `wan_dev` | WAN interface name |
| `apd_dev` | Alternate path device (e.g. WiFi) |
| `apd_cidr` | CIDR for alternate path device |
| `tun_dev` | TUN device name (default `xray0`) |
| `xray_out_mark` | fwmark for xray outbound traffic |
| `xray_traffic_mark` | fwmark for tunneled traffic |
| `transmission_whitelist` | Transmission remote whitelist |
| `dhcp_range` | dnsmasq DHCP range |
| `reality_pbk` | REALITY public key |
| `reality_sni` | REALITY SNI |
| `reality_sid` | REALITY short ID |

### xray_server inventory vars

| Var | Description |
|---|---|
| `reality_dest` | REALITY destination (e.g. `example.com:443`) |
| `reality_server_names` | Allowed REALITY server names |
| `reality_short_id` | REALITY short ID |

### xray_proxy inventory vars

| Var | Description |
|---|---|
| `xray_server_addr` | xray_server IP for nginx stream proxy |
| `proxy_domain` | Public domain for subsrv (TLS via certbot) |
| `reality_pbk` | REALITY public key |
| `reality_sni` | REALITY SNI |
| `reality_sid` | REALITY short ID |

## CLI reference

```
xrayvpn start              restart tunnel
xrayvpn stop               stop tunnel
xrayvpn refresh            refresh geodata and CIDRs

xrayvpn link list          show all links
xrayvpn link init ...      initialize with VLESS params
xrayvpn link add <url>     add VLESS URL  [--rotate]
xrayvpn link remove <id>   remove link
xrayvpn link choose <id>   set active link
```

## Admin panel

Available at `https://<proxy_domain>/admin/`. Manages subscription links — enable/disable, per-link comments, device tracking, and QR codes for each config.
