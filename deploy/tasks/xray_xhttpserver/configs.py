from pyinfra.operations import files, server
from deploy.triggers import notify

server.group(
    name="Create xray-cert group",
    group="xray-cert", system=True)

server.user(
    name="Create xray system user",
    user="xray", system=True, shell="/usr/sbin/nologin",
    home="/nonexistent", create_home=False, ensure_home=False,
    groups=["xray-cert"])

server.shell(
    name="Grant xray-cert group access to Let's Encrypt certs",
    commands=[
        "chgrp -R xray-cert /etc/letsencrypt/live /etc/letsencrypt/archive",
        "chmod g+rx /etc/letsencrypt/live /etc/letsencrypt/archive",
        "find /etc/letsencrypt/archive -name 'privkey*' -exec chmod g+r {} +",
    ])

files.directory(
    name="Ensure /var/www/html exists",
    path="/var/www/html",
    present=True, mode="0755", user="www-data", group="www-data")

notify("xray", files.template(
    name="Deploy xray config",
    src="deploy/templates/xray_xhttpserver/xray-config.json.j2",
    dest="/usr/local/etc/xray/config.json",
    mode="0640", user="root", group="xray"))

notify("xray", files.directory(
    name="Create xray.service.d",
    path="/etc/systemd/system/xray.service.d",
    mode="0755", user="root", group="root"))

notify("xray", files.template(
    name="Deploy xray service override",
    src="deploy/templates/xray_xhttpserver/xray-override.conf.j2",
    dest="/etc/systemd/system/xray.service.d/override.conf",
    mode="0644", user="root", group="root"))

notify("nftables", files.template(
    name="Deploy /etc/nftables.conf",
    src="deploy/templates/xray_xhttpserver/nftables.conf.j2",
    dest="/etc/nftables.conf",
    mode="0644", user="root", group="root"))

notify("ssh", files.template(
    name="Deploy /etc/ssh/sshd_config",
    src="deploy/templates/xray_xhttpserver/sshd_config.j2",
    dest="/etc/ssh/sshd_config",
    mode="0644", user="root", group="root"))

notify("nginx", files.template(
    name="Deploy nginx default site",
    src="deploy/templates/xray_xhttpserver/nginx-default.conf.j2",
    dest="/etc/nginx/sites-available/default",
    mode="0644", user="root", group="root"))

files.template(
    name="Deploy sysctl config",
    src="deploy/templates/xray_xhttpserver/sysctl.conf.j2",
    dest="/etc/sysctl.d/99-xray.conf",
    mode="0644", user="root", group="root")

server.shell(
    name="Apply sysctl tuning",
    commands=["sysctl -p /etc/sysctl.d/99-xray.conf"])

notify("clientrotate", files.template(
    name="Deploy clientrotate.service",
    src="deploy/templates/xray_xhttpserver/clientrotate.service.j2",
    dest="/etc/systemd/system/clientrotate.service",
    mode="0644", user="root", group="root"))

notify("clientrotate", files.template(
    name="Deploy clientrotate.timer",
    src="deploy/templates/xray_xhttpserver/clientrotate.timer.j2",
    dest="/etc/systemd/system/clientrotate.timer",
    mode="0644", user="root", group="root"))
