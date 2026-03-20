from pyinfra.operations import files

from operations.triggers import notify

notify("networking", files.template(name="Deploy /etc/network/interfaces", src="templates/interfaces.j2", dest="/etc/network/interfaces", mode="0644", user="root", group="root"))
notify("nftables", files.template(name="Deploy /etc/nftables.conf", src="templates/nftables.conf.j2", dest="/etc/nftables.conf", mode="0644", user="root", group="root"))
notify("dnsmasq", files.template(name="Deploy /etc/dnsmasq.conf", src="templates/dnsmasq.conf.j2", dest="/etc/dnsmasq.conf", mode="0644", user="root", group="root"))
notify("hostapd", files.template(name="Deploy /etc/hostapd/hostapd.conf", src="templates/hostapd.conf.j2", dest="/etc/hostapd/hostapd.conf", mode="0600", user="root", group="root"))
notify("sshd", files.template(name="Deploy /etc/ssh/sshd_config", src="templates/sshd_config.j2", dest="/etc/ssh/sshd_config", mode="0644", user="root", group="root"))
notify("navidrome", files.template(name="Deploy /etc/navidrome/navidrome.toml", src="templates/navidrome.toml.j2", dest="/etc/navidrome/navidrome.toml", mode="0644", user="navidrome", group="navidrome"))

for d in ["/srv/navidrome/music", "/srv/navidrome/data", "/srv/navidrome/cache"]:
    notify("navidrome", files.directory(name=f"Create {d}", path=d, mode="0755", user="navidrome", group="navidrome"))

notify("nftables", files.directory(name="Create nftables.service.d", path="/etc/systemd/system/nftables.service.d", mode="0755", user="root", group="root"))
notify("nftables", files.put(
    name="Deploy nftables-override.conf",
    src="templates/nftables-override.conf",
    dest="/etc/systemd/system/nftables.service.d/override.conf",
    mode="0644",
    user="root",
    group="root",
))
