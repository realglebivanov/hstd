# hstd Deployment Guide

> This project is for educational purposes only. It is not intended for use in circumventing lawful restrictions or enabling unauthorized access to networks or services.

This repository currently has four deployable roles:

1. `xray_server`: the required public VLESS+REALITY server.
2. `xray_proxy`: the required TCP proxy for `xray_server`, plus the HTTPS subscription/admin service.
3. `xray_xhttpserver`: an optional VLESS+XHTTP+TLS server meant to sit behind a CDN.
4. `hstd`: an optional local Debian router/client that runs `xrayvpnd`, Wi-Fi AP services, Transmission, and Navidrome.

`inventories/all.py` deploys all four roles. The minimal public deployment is `xray_server` plus `xray_proxy`.

## Topology

```text
                         Internet
                            |
          +-----------------+------------------+
          |                 |                  |
     xray_server       xray_proxy      xray_xhttpserver
   VLESS+REALITY      nginx stream    VLESS+XHTTP+TLS
     required           required          optional
      :443/tcp         :443/tcp             :443/tcp
      :80/http         subsrv :8080         :80/http
                           |
                    subscription/admin
                    https://<proxy_domain>:8080/

                            |
                          hstd
                local router/client (optional)
      xrayvpnd + tun2socksd + dnsmasq + hostapd + Navidrome
```

`subsrv` exposes server configs defined in the `servers` list of `subsrv_config` in `inventories/xray_proxy.py`. The default configuration provides three entries:

1. Direct REALITY to `xray_server_addr`.
2. Proxied REALITY to `xray_proxy_addr`.
3. CDN-fronted XHTTP/TLS to `xhttp_cdn_domain` if you deploy the optional `xray_xhttpserver` path.

Important: `proxy_domain` is only for the HTTPS subscription/admin service on port `8080`. The proxied REALITY config itself targets the proxy IP address from `xray_proxy_addr`.

## Prerequisites

### Local deploy machine

- Linux is recommended.
- `inventories/all.py` builds a Debian package locally with `make deb`, so you need `dpkg-deb` if you deploy `hstd`.
- `asdf`
- `make`
- `pass`
- `gpg`
- `ssh`
- `curl`
- Python and Go matching [`.tool-versions`](./.tool-versions)
- Python packages: `pyinfra` and `bcrypt`

If you only deploy the required public roles (`xray_server` and `xray_proxy`), macOS can work as long as you have the required Python/Go toolchain. For the optional `hstd` deployment, a Linux workstation is the safest choice because of the `.deb` build step.

### Remote machines

- Debian 12+ on each remote host
- 2 required Internet-facing hosts for `xray_server` and `xray_proxy`
- 1 optional Internet-facing host for `xray_xhttpserver`
- 1 optional local/LAN Debian host for `hstd`
- A domain for the subscription/admin service
- A separate origin domain plus a CDN hostname if you deploy `xray_xhttpserver`

## Install Local Tooling

Install the exact Go and Python versions from the repo root:

```bash
asdf plugin add golang
asdf plugin add python
asdf install
```

Install the Python packages used by deployment:

```bash
python -m pip install pyinfra bcrypt
```

Make sure `pass` works before continuing:

```bash
pass ls
```

## DNS and CDN Layout

### Required

| Purpose | Example | Points to |
|---|---|---|
| `proxy_domain` | `x.example.com` | `xray_proxy_addr` |

Notes:

- `proxy_domain` gets a Let's Encrypt certificate for `subsrv` on `xray_proxy`.

### Optional `xray_xhttpserver` path

If you deploy `xray_xhttpserver`, also configure:

| Purpose | Example | Points to |
|---|---|---|
| `xhttp_source_domain` | `cdn.example.com` | `xray_xhttpserver_addr` |
| `xhttp_cdn_domain` | `pub.cdn.example.com` | your CDN resource that fronts `xhttp_source_domain` |

Notes:

- `xhttp_source_domain` gets a Let's Encrypt certificate on `xray_xhttpserver`.
- `xhttp_cdn_domain` is what subscription clients use for the third config.
- `xhttp_source_domain` must resolve directly to `xray_xhttpserver` during ACME validation.

## Generate Secrets

### REALITY key pair

Use any Xray binary that supports `x25519`. One way is:

```bash
wget https://github.com/XTLS/Xray-core/releases/latest/download/Xray-linux-64.zip
unzip Xray-linux-64.zip xray
chmod +x xray
./xray x25519
```

Save:

- the public key for `deploy/xray.py`
- the private key for `pass`

### Rotation secret

```bash
openssl rand -hex 32
```

### REALITY short ID

```bash
openssl rand -hex 8
```

### `pass` entries

The deployment code reads these exact entries:

```bash
pass insert hstd/rotate_secret
pass insert hstd/xray_server/reality_private_key
pass insert hstd/xray_proxy/sub_path
pass insert hstd/xray_proxy/admin_user
pass insert hstd/xray_proxy/admin_password
```

If you deploy `hstd`, also add:

```bash
pass insert hstd/wpa_passphrase
```

Notes:

- `admin_password` is stored as plain text in `pass`; deployment hashes it with `bcrypt` automatically.
- `sub_path` is still required. It acts as the legacy subscription path for link index `0`, even though the admin UI also exposes encoded per-link URLs.

## Edit Repository Config

### 1. Update `deploy/xray.py`

Replace the sample values with your own:

```python
xray_version = "26.3.27"

xray_xhttpserver_addr = "YOUR_XHTTPSERVER_IP"
xray_server_addr = "YOUR_XRAY_SERVER_IP"
xray_proxy_addr = "YOUR_XRAY_PROXY_IP"

reality_pbk = "YOUR_REALITY_PUBLIC_KEY"
reality_sni = "example.com"
reality_sid = "YOUR_REALITY_SHORT_ID"

proxy_domain = "x.example.com"
xhttp_source_domain = "cdn.example.com"
xhttp_cdn_domain = "pub.cdn.example.com"
xhttp_path = "/images/625385d043bfac1b"

letsencrypt_email = "you@example.com"
```

These values are imported by inventory files and used to populate host variables. What each field is used for:

- `xray_version`: version of xray-core installed on `xray_server` and `xray_xhttpserver`
- `xray_server_addr`: direct REALITY server address
- `xray_proxy_addr`: TCP proxy IP used by the proxied REALITY config
- `xray_xhttpserver_addr`: XHTTP origin server IP for the optional CDN-fronted path
- `reality_pbk`: public half of the REALITY key pair
- `reality_sni`: REALITY SNI and destination hostname base
- `reality_sid`: short ID advertised to clients
- `proxy_domain`: hostname for the HTTPS subscription/admin service
- `xhttp_source_domain`: certificate/origin hostname on the optional `xray_xhttpserver`
- `xhttp_cdn_domain`: CDN hostname given to clients for the optional XHTTP config
- `xhttp_path`: XHTTP path used by the optional XHTTP server and client config
- `letsencrypt_email`: email for Let's Encrypt certificate registration

### 2. Update inventory files

Review these files and change the host addresses, SSH users, and domains:

- `inventories/xray_server.py`
- `inventories/xray_proxy.py`
- `inventories/xray_xhttpserver.py` if you deploy the optional CDN/XHTTP path
- `inventories/hstd.py` if you deploy the local router/client

Current inventory-specific knobs:

#### `inventories/xray_server.py`

- `ssh_user`
- `reality_dest`
- `reality_server_names`
- `reality_short_id`
- `reality_private_key`
- `rotate_secret`

`reality_dest`, `reality_server_names`, and `reality_short_id` are derived from `deploy/xray.py` by default. `reality_private_key` and `rotate_secret` come from `pass`.

#### `inventories/xray_proxy.py`

- `ssh_user`
- `_sudo`
- `xray_server_addr`
- `rotate_secret`
- `proxy_domain`
- `letsencrypt_email`
- `sub_path`
- `admin_user`
- `admin_password_hash`
- `subsrv_config`

`proxy_domain` and `letsencrypt_email` come from `deploy/xray.py`. The other non-secret values also come from `deploy/xray.py`, and the secrets come from `pass`.

`subsrv_config` is a dict that gets deployed as `/etc/subsrv/config.json`. It defines the server entries and client routing rules that `subsrv` serves to subscribers. The structure contains:

- `servers`: list of server entries, each with `remark`, `host`, and either REALITY fields (`realityPbk`, `realitySni`, `realitySid`) or XHTTP fields (`xhttpPath`)
- `routingRules`: list of Xray routing rules included in generated client configs

#### `inventories/xray_xhttpserver.py`

- `ssh_user`
- `rotate_secret`
- `xhttp_source_domain`
- `xhttp_path`
- `letsencrypt_email`

#### `inventories/hstd.py`

If you use `hstd`, audit these fields carefully:

- host IP / `ssh_user` / `_sudo`
- `wpa_passphrase`
- `apd_ip`
- `apd_cidr`
- `apd_gateway_cidr`
- `lan_gateway_cidr`
- `dhcp_range_start`
- `dhcp_range_end`
- `transmission_rpc_whitelist`
- `xray_out_mark`
- `xray_traffic_mark`
- `wan_dev`
- `apd_dev`
- `lan_dev`
- `tun_dev`

The `hstd` role no longer carries REALITY or xray server parameters directly. Instead, it adds a subscription URL during deploy and uses `xrayvpn sub sync` to fetch connections from `subsrv`.

### 3. Review hardcoded repo-specific defaults

Several important values are still hardcoded outside the inventories:

- `deploy/roles/hstd.py`: creates user `gleb` and installs a hardcoded SSH public key
- `deploy/templates/hstd/hostapd.conf.j2`: SSID, channel plan, and `country_code=RU`

If those do not match your environment, change them before deploying.

## Optional Local Build Checks

The deploy tasks build artifacts automatically, but these commands are useful as a preflight check:

```bash
cd xrayvpn
make xrayconnectord
make deb
```

Current outputs:

- `make xrayconnectord`
  - `xrayvpn/target/clientrotate`
  - `xrayvpn/target/subsrv`
- `make build`
  - `xrayvpn/target/xrayvpnd`
  - `xrayvpn/target/xrayvpn`
  - `xrayvpn/target/tun2socksd`
- `make deb` (depends on `build`)
  - all of the above plus `xrayvpn/target/deb/xrayvpn_0.1.0_amd64.deb`

`make deb` is only needed for the optional `hstd` role. The required public roles only need `make xrayconnectord`.

## Deploy

### Full stack

```bash
pyinfra inventories/all.py deploy.py
```

Remember: `inventories/all.py` includes `hstd`. If you do not want that role, do not use `all.py`.

### Required public roles

```bash
pyinfra inventories/xray_server.py deploy.py
pyinfra inventories/xray_proxy.py deploy.py
```

### Optional roles

```bash
pyinfra inventories/xray_xhttpserver.py deploy.py
pyinfra inventories/hstd.py deploy.py
```

### What deployment does

#### `xray_server`

- installs `nftables`, `nginx`, `curl`, `ca-certificates`
- installs `xray-core` version `26.3.27`
- uploads `/usr/local/etc/xray/config.json`
- uploads `clientrotate` and its systemd units
- enables `nftables`, `nginx`, `xray`, `ssh`, `clientrotate`, and `clientrotate.timer`
- runs `clientrotate.service` once after deploy

#### `xray_proxy`

- installs `nftables`, `nginx`, `libnginx-mod-stream`, `curl`, `certbot`
- uploads `subsrv`
- deploys `/etc/subsrv/config.json` with server entries and routing rules from `subsrv_config`
- configures nginx stream proxy on `:443`
- runs `subsrv` directly on HTTPS `:8080`
- obtains a Let's Encrypt certificate for `proxy_domain`
- enables `nftables`, `nginx`, `ssh`, `subsrv`, and `certbot.timer`

#### `xray_xhttpserver` (optional)

- installs `nftables`, `nginx`, `curl`, `ca-certificates`, `certbot`
- installs `xray-core` version `26.3.27`
- uploads `/usr/local/etc/xray/config.json`
- uploads `clientrotate` and its systemd units
- obtains a Let's Encrypt certificate for `xhttp_source_domain`
- enables `nftables`, `nginx`, `xray`, `ssh`, `clientrotate`, and `clientrotate.timer`
- enables `certbot.timer`

#### `hstd` (optional)

- builds and uploads `xrayvpn_0.1.0_amd64.deb`
- installs `nftables`, `dnsmasq`, `hostapd`, `transmission-daemon`, `ffmpeg`, `curl`, `rsync`, `networkd-dispatcher`, `systemd-resolved`, `dns-root-data`, and `firmware-misc-nonfree`
- downloads and installs Navidrome `0.60.3`
- configures `systemd-networkd`, nftables, dnsmasq, hostapd, Transmission, Navidrome, and the `xrayvpn` systemd overrides
- adds a subscription URL pointing at `subsrv` and runs `xrayvpn sub sync` to fetch initial connections
- runs `clientrotate.service` to perform an initial client rotation
- enables `xrayvpnd`, `nftables`, `dnsmasq`, `hostapd`, `navidrome`, `transmission-daemon`, `ssh`, `systemd-networkd`, `networkd-dispatcher`, `systemd-resolved`, `clientrotate.timer`, and related services

## Verify

### `xray_server`

```bash
systemctl status xray
systemctl status nginx
systemctl status nftables
systemctl status clientrotate.timer
```

### `xray_proxy`

```bash
systemctl status nginx
systemctl status subsrv
systemctl status nftables
systemctl status certbot.timer
```

Test the legacy subscription path:

```bash
curl https://<proxy_domain>:8080/<sub_path>
```

Expected result: a JSON array containing the direct and proxied REALITY client configs, plus the optional XHTTP config when that path is enabled.

Open the admin UI:

```text
https://<proxy_domain>:8080/admin/
```

Use the username from `hstd/xray_proxy/admin_user` and the plain-text password from `hstd/xray_proxy/admin_password`.

### `xray_xhttpserver` (if deployed)

```bash
systemctl status xray
systemctl status nginx
systemctl status nftables
systemctl status certbot.timer
systemctl status clientrotate.timer
```

### `hstd` (if deployed)

```bash
systemctl status xrayvpnd
systemctl status tun2socksd
systemctl status xrayvpnd-refresh.timer
systemctl status clientrotate.timer
systemctl status dnsmasq
systemctl status hostapd
systemctl status transmission-daemon
systemctl status navidrome
systemctl status systemd-networkd
systemctl status networkd-dispatcher
```

Useful local checks on `hstd`:

```bash
xrayvpn conn list
xrayvpn sub list
xrayvpn start
xrayvpn stop
xrayvpn refresh
```

## How the Current Project Behaves

### Subscription/admin service

`subsrv` listens on `:8080` with HTTPS. It is not behind nginx. It reads its server entries and routing rules from `/etc/subsrv/config.json` (deployed from `subsrv_config` in the inventory). It serves:

- `GET /admin/`: admin page (cookie-based session auth)
- `GET /admin/ws`: websocket feed for admin updates
- `GET /{link}`: JSON subscription response

The admin page currently manages a fixed pool of `100` link slots. It can:

- enable or disable a link
- edit a comment
- show the encoded URL and QR code for each slot
- track devices seen in the last 24 hours

`subsrv` also periodically refreshes Russian CIDR data (every 2 hours) and injects it into the `cidr:ru` routing rules in generated client configs.

### UUID rotation

The project derives client UUIDs deterministically from:

- `rotate_secret`
- link index
- the current day in UTC with a `-3h` offset before truncation

The server-side `clientrotate` binaries write the current UUID set into the Xray configs on `xray_server` and on `xray_xhttpserver` when that optional role is deployed. `subsrv` derives the matching UUIDs on demand, so clients that refresh their subscription get the right values automatically.

Each rotation produces 200 UUIDs: 100 for the current day and 100 for the previous day, ensuring a graceful transition window.

### Split routing in subscription client configs

The JSON configs returned by `subsrv` include routing rules defined in the `routingRules` field of `/etc/subsrv/config.json` (configured via `subsrv_config` in the inventory). This split routing is separate from the Linux-level split routing on `hstd`.

Each generated client config currently contains:

- a local SOCKS inbound on `127.0.0.1:10808`
- a `direct` outbound
- a `proxy` outbound using one of the advertised server entries
- a `block` outbound

The default routing rules in the inventory currently do this:

- force `domain:yonote.ru` and `domain:hstd.space` through the proxy
- send `geoip:ru`, `geoip:private`, and `cidr:ru` directly
- send `geosite:category-ru` and `geosite:category-gov-ru` directly
- send all remaining `tcp` and `udp` traffic through the proxy

These rules can be changed by editing the `routingRules` list in `inventories/xray_proxy.py` and redeploying.

### `hstd`

The `hstd` host consumes the HTTPS subscription endpoint. During deploy, a subscription URL is added via `xrayvpn sub add`, and `clientrotate.service` runs `xrayvpn sub sync` to fetch connections from `subsrv`. The `clientrotate.timer` periodically re-syncs to pick up rotated UUIDs.

### Included traffic-splitting rules on `hstd`

When the tunnel is up, `tun2socksd` and `xrayvpnd` install a split-routing setup with these behaviors:

- The main routing table's default route is replaced with the TUN device, so ordinary traffic enters `xray0`.
- A separate direct route table keeps the original WAN gateway for traffic that must bypass the tunnel.
- Packets emitted by xray-core are marked with `xray_out_mark` and forced into the direct table, which prevents routing loops.
- The service users `debian-transmission` and `navidrome` bypass the tunnel at the IP-rule layer and use the direct table for Internet traffic.
- Those same service users keep access to the local APD subnet through a higher-priority rule that sends APD-destined traffic through the main table instead of the direct table.
- `tun2socksd` adds an `xray_vpn` nftables table that marks forwarded tunnel traffic with `xray_traffic_mark` and allows the paths `lo -> tun`, `apd -> tun`, `tun -> wan`, and `tun -> apd`.

Inside `xrayvpnd`, the active outbound selection is also split by protocol and destination:

- BitTorrent traffic goes direct.
- FTP ports `20-21` go direct.
- DNS traffic to `8.8.8.8`, `8.8.4.4`, and `1.1.1.1` goes direct.
- Traffic matching the subscription's routing rules (Russian CIDRs, geoip, geosite) goes direct.
- Remaining `tcp` and `udp` traffic goes through the currently selected VLESS outbound.

In practice, that means:

- local service traffic for Transmission and Navidrome bypasses the VPN
- Russian and private destinations bypass the VPN
- everything else is sent into the tunnel and then proxied through the active VLESS link

## Troubleshooting

### Local deploy fails before connecting anywhere

- Make sure `python -m pip install pyinfra bcrypt` is done.
- Make sure every required `pass` entry exists under `hstd/...`.
- If `make deb` fails, install `dpkg-deb` and related packaging tools or skip the `hstd` role.

### Certbot fails

- `proxy_domain` must resolve to `xray_proxy_addr`.
- `xhttp_source_domain` must resolve directly to `xray_xhttpserver_addr` if you deploy `xray_xhttpserver`.
- Update `letsencrypt_email` in `deploy/xray.py` if needed.

### Subscription works but one of the three configs is broken

- direct REALITY depends on `xray_server_addr`, `reality_pbk`, `reality_sni`, and `reality_sid`
- proxied REALITY depends on `xray_proxy_addr` forwarding TCP `:443` to `xray_server_addr:443`
- XHTTP depends on deploying `xray_xhttpserver`, plus `xhttp_source_domain`, `xhttp_cdn_domain`, `xhttp_path`, and working CDN origin configuration

### `hstd` deploy works but Wi-Fi details are wrong

Edit `deploy/templates/hstd/hostapd.conf.j2`. The SSID, channel settings, and country code are not inventory variables right now.

## Maintenance

- Change server IPs, domains, and REALITY parameters in `deploy/xray.py`, then redeploy the affected roles.
- Change SSH users and role-specific host data in the inventory files, then redeploy.
- Certificates renew automatically via `certbot.timer` on `xray_proxy` and on `xray_xhttpserver` if deployed.
- `xrayvpn refresh` refreshes geodata on `hstd`.
- If you rotate the REALITY key pair, update both `deploy/xray.py` and `hstd/xray_server/reality_private_key`, then redeploy `xray_server`, `xray_proxy`, and `hstd` if used.
