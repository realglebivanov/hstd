from pyinfra.operations import files
from deploy.triggers import notify

for src, dest in [
    ("deploy/templates/hstd/network/10-apd.network.j2", "/etc/systemd/network/10-apd.network"),
    ("deploy/templates/hstd/network/20-wan.network.j2", "/etc/systemd/network/20-wan.network"),
    ("deploy/templates/hstd/network/30-lan.network.j2", "/etc/systemd/network/30-lan.network"),
]:
    notify("systemd-networkd", files.template(
        name=f"Deploy {dest}",
        src=src, dest=dest,
        mode="0644", user="root", group="root"))

notify("nftables", files.template(
        name="Deploy /etc/nftables.conf",
        src="deploy/templates/hstd/nftables.conf.j2",
        dest="/etc/nftables.conf",
        mode="0644", user="root", group="root"))

notify("dnsmasq", files.template(
        name="Deploy /etc/dnsmasq.conf",
        src="deploy/templates/hstd/dnsmasq.conf.j2",
        dest="/etc/dnsmasq.conf",
        mode="0644", user="root", group="root"))

notify("transmission-daemon", files.template(
        name="Deploy /etc/transmission-daemon/settings.json",
        src="deploy/templates/hstd/transmission-settings.json",
        dest="/etc/transmission-daemon/settings.defaults.json",
        mode="0644", user="root", group="root"))

notify("transmission-daemon", files.directory(
        name="Create transmission-daemon.service.d",
        path="/etc/systemd/system/transmission-daemon.service.d",
        mode="0755", user="root", group="root"))

notify("transmission-daemon", files.put(
        name="Deploy transmission-daemon override.conf",
        src="deploy/templates/hstd/transmission-daemon-override.conf",
        dest="/etc/systemd/system/transmission-daemon.service.d/override.conf",
        mode="0644", user="root", group="root"))

notify("hostapd", files.template(
    name="Deploy /etc/hostapd/hostapd.conf",
    src="deploy/templates/hstd/hostapd.conf.j2",
    dest="/etc/hostapd/hostapd.conf",
    mode="0600", user="root", group="root"))

notify("ssh", files.template(
    name="Deploy /etc/ssh/sshd_config",
    src="deploy/templates/hstd/sshd_config.j2",
    dest="/etc/ssh/sshd_config",
    mode="0644", user="root", group="root"))

notify("navidrome", files.template(
    name="Deploy /etc/navidrome/navidrome.toml",
    src="deploy/templates/hstd/navidrome.toml.j2",
    dest="/etc/navidrome/navidrome.toml",
    mode="0644", user="navidrome", group="navidrome"))

for d in ["/srv/navidrome/music", "/srv/navidrome/data", "/srv/navidrome/cache"]:
    notify("navidrome", files.directory(
        name=f"Create {d}", path=d, mode="0755", user="navidrome", group="navidrome"))

notify("transmission-daemon", files.directory(
    name="Create /srv/transmission",
    path="/srv/transmission",
    mode="0755", user="root", group="root"))

notify("transmission-daemon", files.directory(
    name="Create /srv/transmission/downloads",
    path="/srv/transmission/downloads",
    mode="0755", user="debian-transmission", group="debian-transmission"))

notify("transmission-daemon", files.directory(
    name="Create /srv/transmission/downloads/music",
    path="/srv/transmission/downloads/music",
    mode="2755", user="debian-transmission", group="navidrome"))

notify("transmission-daemon", files.directory(
    name="Create /srv/transmission/incomplete",
    path="/srv/transmission/incomplete",
    mode="0750", user="debian-transmission", group="debian-transmission"))

notify("navidrome", files.link(
    name="Link Transmission music into Navidrome library",
    path="/srv/navidrome/music/Transmission",
    target="/srv/transmission/downloads/music",
    user="navidrome", group="navidrome"))

notify("networkd-dispatcher", files.template(
    name="Deploy networkd-dispatcher routable.d script",
    src="deploy/templates/hstd/networkd-dispatcher-routable.j2",
    dest="/etc/networkd-dispatcher/routable.d/xrayvpnd",
    mode="0755", user="root", group="root"))

for state in ["no-carrier", "off", "degraded"]:
    notify("networkd-dispatcher", files.template(
        name=f"Deploy networkd-dispatcher {state}.d script",
        src="deploy/templates/hstd/networkd-dispatcher-no-carrier.j2",
        dest=f"/etc/networkd-dispatcher/{state}.d/xrayvpnd",
        mode="0755", user="root", group="root"))

notify("nftables", files.directory(
    name="Create nftables.service.d",
    path="/etc/systemd/system/nftables.service.d",
    mode="0755", user="root", group="root"))

notify("nftables", files.put(
    name="Deploy nftables override.conf",
    src="deploy/templates/hstd/nftables-override.conf",
    dest="/etc/systemd/system/nftables.service.d/override.conf",
    mode="0644", user="root", group="root"))

notify("xrayvpnd", files.directory(
    name="Create xrayvpnd.service.d",
    path="/etc/systemd/system/xrayvpnd.service.d",
    mode="0755", user="root", group="root"))

notify("xrayvpnd", files.template(
    name="Deploy xrayvpnd override.conf",
    src="deploy/templates/hstd/xrayvpnd-override.conf.j2",
    dest="/etc/systemd/system/xrayvpnd.service.d/override.conf",
    mode="0644", user="root", group="root"))

notify("xrayvpnd", files.directory(
    name="Create tun2socksd.service.d",
    path="/etc/systemd/system/tun2socksd.service.d",
    mode="0755", user="root", group="root"))

notify("xrayvpnd", files.template(
    name="Deploy tun2socksd override.conf",
    src="deploy/templates/hstd/tun2socksd-override.conf.j2",
    dest="/etc/systemd/system/tun2socksd.service.d/override.conf",
    mode="0644", user="root", group="root"))

notify("clientrotate", files.template(
    name="Deploy clientrotate.service",
    src="deploy/templates/hstd/clientrotate.service.j2",
    dest="/etc/systemd/system/clientrotate.service",
    mode="0644", user="root", group="root"))

notify("clientrotate", files.template(
    name="Deploy clientrotate.timer",
    src="deploy/templates/hstd/clientrotate.timer.j2",
    dest="/etc/systemd/system/clientrotate.timer",
    mode="0644", user="root", group="root"))
