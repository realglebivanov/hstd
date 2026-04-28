import hashlib
import subprocess

from pyinfra.operations import apt, files, server
from pyinfra import host
from pyinfra.facts.server import Command
from pyinfra.facts.files import Sha256File
from deploy.triggers import notify
from deploy.xray import xray_version

_APT_ENV = {"DEBIAN_FRONTEND": "noninteractive"}

apt.update(name="Update apt cache", cache_time=3600, _env=_APT_ENV)
for pkg in [
    "nftables",
    "nginx",
    "curl",
    "ca-certificates",
]: notify(pkg, apt.packages(
    name=f"Install {pkg}", packages=[pkg], present=True, _env=_APT_ENV))

server.user(
    name="Create xray system user",
    user="xray", system=True, shell="/usr/sbin/nologin",
    home="/nonexistent", create_home=False, ensure_home=False)

installed_version = host.get_fact(Command, "xray version 2>/dev/null | head -1 | awk '{print $2}'")
unit_user = (host.get_fact(Command, "grep -E '^User=' /etc/systemd/system/xray.service 2>/dev/null | tail -1 | cut -d= -f2") or "").strip()
if installed_version != xray_version or unit_user != "xray":
    notify("xray", server.shell(
        name=f"Install xray-core v{xray_version} as xray user",
        commands=[f'bash -c "$(curl -fsSL https://github.com/XTLS/Xray-install/raw/main/install-release.sh)" @ install --version {xray_version} --install-user xray --force']))

CLIENTROTATE_LOCAL = "xrayvpn/target/clientrotate"
CLIENTROTATE_REMOTE = "/usr/local/bin/clientrotate"
subprocess.run(["make", "xrayconnectord"], cwd="xrayvpn", check=True)
clientrotate_sha256 = hashlib.sha256(open(CLIENTROTATE_LOCAL, "rb").read()).hexdigest()
if host.get_fact(Sha256File, path=CLIENTROTATE_REMOTE) != clientrotate_sha256:
    notify("clientrotate", files.put(
        name="Upload clientrotate binary",
        src=CLIENTROTATE_LOCAL,
        dest=CLIENTROTATE_REMOTE, mode="0755"))
