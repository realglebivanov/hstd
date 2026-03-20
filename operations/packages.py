import hashlib
import subprocess
import urllib.request

from pyinfra import host
from pyinfra.operations import apt, files, server
from pyinfra.facts.files import Sha256File

from operations.triggers import notify

DEB_PATH = "godeps/target/deb/xrayvpn_0.1.0_amd64.deb"
REMOTE_DEB_PATH = "/tmp/xrayvpn_0.1.0_amd64.deb"

NAVIDROME_VERSION = "0.60.3"
NAVIDROME_URL = f"https://github.com/navidrome/navidrome/releases/download/v{NAVIDROME_VERSION}/navidrome_{NAVIDROME_VERSION}_linux_amd64.deb"
NAVIDROME_CHECKSUMS_URL = f"https://github.com/navidrome/navidrome/releases/download/v{NAVIDROME_VERSION}/navidrome_checksums.txt"
NAVIDROME_DEB_PATH = f"/tmp/navidrome_{NAVIDROME_VERSION}_linux_amd64.deb"

_APT_ENV = {"DEBIAN_FRONTEND": "noninteractive"}

subprocess.run(["make", "deb"], cwd="godeps", check=True)

apt.update(name="Update apt cache", cache_time=3600, _env=_APT_ENV)

for pkg in [
    "nftables",
    "dnsmasq",
    "hostapd",
    "ffmpeg",
    "curl",
    "rsync",
    "networkd-dispatcher"
]: notify(pkg, apt.packages(name=f"Install {pkg}", packages=[pkg], present=True, _env=_APT_ENV))

local_deb_sha256 = hashlib.sha256(open(DEB_PATH, "rb").read()).hexdigest()
if host.get_fact(Sha256File, path=REMOTE_DEB_PATH) != local_deb_sha256:
    files.put(name="Upload xrayvpn .deb package", src=DEB_PATH, dest=REMOTE_DEB_PATH, mode="0644")
    notify("xrayvpnd", server.shell(
        name="Install xrayvpn .deb package",
        commands=[f"dpkg -i {REMOTE_DEB_PATH}"], _env=_APT_ENV))

checksums_txt = urllib.request.urlopen(NAVIDROME_CHECKSUMS_URL).read().decode()
navidrome_deb_name = f"navidrome_{NAVIDROME_VERSION}_linux_amd64.deb"
navidrome_sha256 = next(
    line.split()[0] for line in checksums_txt.splitlines()
    if line.endswith(navidrome_deb_name)
)
if host.get_fact(Sha256File, path=NAVIDROME_DEB_PATH) != navidrome_sha256:
    server.shell(
        name="Download navidrome .deb package",
        commands=[f"curl -fsSL -o {NAVIDROME_DEB_PATH} {NAVIDROME_URL}"])
    notify("navidrome", server.shell(
        name="Install navidrome .deb package",
        commands=[f"dpkg -i {NAVIDROME_DEB_PATH}"], _env=_APT_ENV))
