import subprocess

from pyinfra import host
from pyinfra.operations import apt, files, server
from pyinfra.facts.files import File as FileFact

from operations.triggers import notify

DEB_PATH = "godeps/target/deb/xrayvpn_0.1.0_amd64.deb"
REMOTE_DEB_PATH = "/tmp/xrayvpn_0.1.0_amd64.deb"
NAVIDROME_URL = "https://github.com/navidrome/navidrome/releases/download/v0.60.3/navidrome_0.60.3_linux_amd64.deb"
NAVIDROME_DEB_PATH = "/tmp/navidrome_0.60.3_linux_amd64.deb"

_APT_ENV = {"DEBIAN_FRONTEND": "noninteractive"}

subprocess.run(["make", "deb"], cwd="godeps", check=True)

apt.update(name="Update apt cache", cache_time=3600, _env=_APT_ENV)

for pkg in ["nftables", "dnsmasq", "hostapd", "ffmpeg", "curl", "rsync"]:
    notify(pkg, apt.packages(name=f"Install {pkg}", packages=[pkg], present=True, _env=_APT_ENV))

if not host.get_fact(FileFact, path=REMOTE_DEB_PATH):
    files.put(name="Upload xrayvpn .deb package", src=DEB_PATH, dest=REMOTE_DEB_PATH, mode="0644")
    notify("xrayvpnd", server.shell(name="Install xrayvpn .deb package", commands=[f"dpkg -i {REMOTE_DEB_PATH}"], _env=_APT_ENV))

if not host.get_fact(FileFact, path=NAVIDROME_DEB_PATH):
    server.shell(name="Download navidrome .deb package", commands=[f"curl -fsSL -o {NAVIDROME_DEB_PATH} {NAVIDROME_URL}"])
    notify("navidrome", server.shell(name="Install navidrome .deb package", commands=[f"dpkg -i {NAVIDROME_DEB_PATH}"], _env=_APT_ENV))
