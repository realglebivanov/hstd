import hashlib
import subprocess

from pyinfra.operations import apt, files, server
from pyinfra import host
from pyinfra.facts.files import Sha256File
from deploy.triggers import notify

_APT_ENV = {"DEBIAN_FRONTEND": "noninteractive"}

apt.update(name="Update apt cache", cache_time=3600, _env=_APT_ENV)
for pkg in [
    "nftables",
    "nginx",
    "libnginx-mod-stream",
    "curl",
    "certbot",
]: notify(pkg, apt.packages(
    name=f"Install {pkg}", packages=[pkg], present=True, _env=_APT_ENV))

SUBSRV_LOCAL = "xrayvpn/target/subsrv"
SUBSRV_REMOTE = "/usr/local/bin/subsrv"
subprocess.run(["make", "xrayconnectord"], cwd="xrayvpn", check=True)
subsrv_sha256 = hashlib.sha256(open(SUBSRV_LOCAL, "rb").read()).hexdigest()
if host.get_fact(Sha256File, path=SUBSRV_REMOTE) != subsrv_sha256:
    server.shell(name="Remove old subsrv binary", commands=["rm -f " + SUBSRV_REMOTE])
    notify("subsrv", files.put(
        name="Upload subsrv binary",
        src=SUBSRV_LOCAL,
        dest=SUBSRV_REMOTE, mode="0755"))
