import subprocess
import sys

from pyinfra import host
from pyinfra.operations import apt, files, systemd, server

DEB_PATH = "godeps/target/deb/xrayvpn_0.1.0_amd64.deb"
REMOTE_DEB_PATH = "/tmp/xrayvpn_0.1.0_amd64.deb"
NAVIDROME_URL = "https://github.com/navidrome/navidrome/releases/download/v0.60.3/navidrome_0.60.3_linux_amd64.deb"
NAVIDROME_DEB_PATH = "/tmp/navidrome_0.60.3_linux_amd64.deb"

subprocess.run(["make", "deb"], cwd="godeps", check=True)

apt.update(name="Update apt cache", cache_time=3600, _env={"DEBIAN_FRONTEND": "noninteractive"})
apt.packages(
    name="Install deps",
    packages=["nftables", "dnsmasq", "hostapd", "ffmpeg", "curl", "rsync"],
    present=True,
    _env={"DEBIAN_FRONTEND": "noninteractive"})

from pyinfra.facts.files import File as FileFact

if not host.get_fact(FileFact, path=REMOTE_DEB_PATH):
    files.put(name="Upload xrayvpn .deb package", src=DEB_PATH, dest=REMOTE_DEB_PATH, mode="0644")
server.shell(name="Install xrayvpn .deb package", commands=[f"dpkg -i {REMOTE_DEB_PATH}"], _env={"DEBIAN_FRONTEND": "noninteractive"})

if not host.get_fact(FileFact, path=NAVIDROME_DEB_PATH):
    server.shell(name="Download navidrome .deb package", commands=[f"curl -fsSL -o {NAVIDROME_DEB_PATH} {NAVIDROME_URL}"])
server.shell(name="Install navidrome .deb package", commands=[f"dpkg -i {NAVIDROME_DEB_PATH}"], _env={"DEBIAN_FRONTEND": "noninteractive"})

files.template(name="Deploy /etc/network/interfaces", src="templates/interfaces.j2", dest="/etc/network/interfaces", mode="0644", user="root", group="root")
files.template(name="Deploy /etc/nftables.conf", src="templates/nftables.conf.j2", dest="/etc/nftables.conf", mode="0644", user="root", group="root")
files.template(name="Deploy /etc/dnsmasq.conf", src="templates/dnsmasq.conf.j2", dest="/etc/dnsmasq.conf", mode="0644", user="root", group="root")
files.template(name="Deploy /etc/hostapd/hostapd.conf", src="templates/hostapd.conf.j2", dest="/etc/hostapd/hostapd.conf", mode="0600", user="root", group="root")
files.template(name="Deploy /etc/ssh/sshd_config", src="templates/sshd_config.j2", dest="/etc/ssh/sshd_config", mode="0644", user="root", group="root")
files.template(name="Deploy /etc/navidrome/navidrome.toml", src="templates/navidrome.toml.j2", dest="/etc/navidrome/navidrome.toml", mode="0644", user="navidrome", group="navidrome")

for d in ["/srv/navidrome/music", "/srv/navidrome/data", "/srv/navidrome/cache"]:
    files.directory(name=f"Create {d}", path=d, mode="0755", user="navidrome", group="navidrome")

files.directory(name="Create nftables.service.d", path="/etc/systemd/system/nftables.service.d", mode="0755", user="root", group="root")
files.put(
    name="Deploy nftables-override.conf",
    src="templates/nftables-override.conf",
    dest="/etc/systemd/system/nftables.service.d/override.conf",
    mode="0644",
    user="root",
    group="root",
)

server.user(
    name="Configure gleb user with SSH key",
    user="gleb",
    groups=["xrayvpn"],
    append=True,
    public_keys=[
        "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQChx5QLwUAa7LWQAFai5sGVFKVlFtSO8iEI/7Y3Vonbf/FGz14N6wk71VOK4k+aa9Pr30EMhAqK8mlPmLVIrWZgmxYmTqXNds81VbCWu0HZvql7FyBbCoLKg+HBt9vYiD1AYhLwMG7bMrc/5uXynFJuB+GkbxHNIvpREfe1445JX6xMksDvHnVkelkbbD20+xukOpK8jXBTPdxepsN6mGYb7M+KbK7PdjHawhnTgt/DDPVhyEvxBOcHJB6iNok1Q27OIFtsEjEEI0bSAvQKY3PPBYdbnYqF4PBHA6kYGnQyyMMYQ7jCqX80GyYbHjXBZ3B8SW1ge6L2Q034ZJvgUnObsgiomBU87KA9chG1Aob4yt8KE6sS69UltdoycsRIK5dASA4prHl6/yiG126Fz3EkMdSv5+xjpdLmHlwiXbirGiQ4XP83dbySIhdg7nxoif5oski+/+4pzsnZNCZuXhRN4Qx4jP6JVnCwaR0j0LPk4BEuYj2xxyJYHh2XL7N/Qn8= gleb@local",
    ],
)

for svc in ["nftables", "dnsmasq", "hostapd", "xrayvpnd", "navidrome"]:
    systemd.service(name=f"Enable and start {svc}", service=svc, running=True, enabled=True, restarted=True, daemon_reload=True)
