import hashlib
import subprocess
import urllib.request

from pyinfra import host
from pyinfra.operations import apt, files, server
from pyinfra.facts.files import Sha256File

from deploy.triggers import notify

_APT_ENV = {"DEBIAN_FRONTEND": "noninteractive"}

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

XRAYVPN_DEB_PATH = "xrayvpn/target/deb/xrayvpn_0.1.0_amd64.deb"
XRAYVPN_REMOTE_DEB_PATH = "/tmp/xrayvpn_0.1.0_amd64.deb"
subprocess.run(["make", "deb"], cwd="xrayvpn", check=True)
xrayvpn_sha256 = hashlib.sha256(open(XRAYVPN_DEB_PATH, "rb").read()).hexdigest()
if host.get_fact(Sha256File, path=XRAYVPN_REMOTE_DEB_PATH) != xrayvpn_sha256:
    files.put(
        name="Upload xrayvpn .deb package",
        src=XRAYVPN_DEB_PATH,
        dest=XRAYVPN_REMOTE_DEB_PATH, mode="0644")
    notify("xrayvpnd", server.shell(
        name="Install xrayvpn .deb package",
        commands=[f"dpkg -i {XRAYVPN_REMOTE_DEB_PATH}"], _env=_APT_ENV))

NAVIDROME_VERSION = "0.60.3"
NAVIDROME_URL = f"https://github.com/navidrome/navidrome/releases/download/v{NAVIDROME_VERSION}/navidrome_{NAVIDROME_VERSION}_linux_amd64.deb"
NAVIDROME_CHECKSUMS_URL = f"https://github.com/navidrome/navidrome/releases/download/v{NAVIDROME_VERSION}/navidrome_checksums.txt"
NAVIDROME_DEB_NAME = f"navidrome_{NAVIDROME_VERSION}_linux_amd64.deb"
NAVIDROME_REMOTE_DEB_PATH = f"/tmp/navidrome_{NAVIDROME_VERSION}_linux_amd64.deb"
checksums_txt = urllib.request.urlopen(NAVIDROME_CHECKSUMS_URL).read().decode()
navidrome_sha256 = next(
    line.split()[0] for line in checksums_txt.splitlines()
    if line.endswith(NAVIDROME_DEB_NAME)
)
if host.get_fact(Sha256File, path=NAVIDROME_REMOTE_DEB_PATH) != navidrome_sha256:
    server.shell(
        name="Download navidrome .deb package",
        commands=[f"curl -fsSL -o {NAVIDROME_REMOTE_DEB_PATH} {NAVIDROME_URL}"])
    notify("navidrome", server.shell(
        name="Install navidrome .deb package",
        commands=[f"dpkg -i {NAVIDROME_REMOTE_DEB_PATH}"], _env=_APT_ENV))
