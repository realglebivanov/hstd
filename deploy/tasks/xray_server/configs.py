import json
from io import StringIO

from pyinfra import host
from pyinfra.operations import files, server
from deploy.triggers import notify

files.directory(
    name="Create journald.conf.d",
    path="/etc/systemd/journald.conf.d",
    mode="0755", user="root", group="root")

notify("systemd-journald", files.put(
    name="Deploy journald log retention config",
    src="deploy/templates/xray_server/journald.conf",
    dest="/etc/systemd/journald.conf.d/retention.conf",
    mode="0644", user="root", group="root"))

server.user(
    name="Create xray system user",
    user="xray", system=True, shell="/usr/sbin/nologin",
    home="/nonexistent", create_home=False, ensure_home=False)

notify("xray", files.template(
    name="Deploy xray config",
    src="deploy/templates/xray_server/xray-config.json.j2",
    dest="/usr/local/etc/xray/config.json",
    mode="0640", user="root", group="xray"))

notify("xray", files.directory(
    name="Create xray.service.d",
    path="/etc/systemd/system/xray.service.d",
    mode="0755", user="root", group="root"))

notify("xray", files.template(
    name="Deploy xray service override",
    src="deploy/templates/xray_server/xray-override.conf.j2",
    dest="/etc/systemd/system/xray.service.d/override.conf",
    mode="0644", user="root", group="root"))

notify("nftables", files.template(
    name="Deploy /etc/nftables.conf",
    src="deploy/templates/xray_server/nftables.conf.j2",
    dest="/etc/nftables.conf",
    mode="0644", user="root", group="root"))

notify("ssh", files.template(
    name="Deploy /etc/ssh/sshd_config",
    src="deploy/templates/xray_server/sshd_config.j2",
    dest="/etc/ssh/sshd_config",
    mode="0644", user="root", group="root"))

notify("nginx", files.template(
    name="Deploy nginx default site",
    src="deploy/templates/xray_server/nginx-default.conf.j2",
    dest="/etc/nginx/sites-available/default",
    mode="0644", user="root", group="root"))

files.template(
    name="Deploy sysctl config",
    src="deploy/templates/xray_server/sysctl.conf.j2",
    dest="/etc/sysctl.d/99-xray.conf",
    mode="0644", user="root", group="root")

server.shell(
    name="Apply sysctl tuning",
    commands=["sysctl -p /etc/sysctl.d/99-xray.conf"])

files.directory(
    name="Create /etc/clientrotate",
    path="/etc/clientrotate",
    present=True, mode="0755", user="root", group="root")

notify("clientrotate", files.put(
    name="Deploy clientrotate config",
    src=StringIO(json.dumps({"routingRules": host.data.routing_rules})),
    dest="/etc/clientrotate/config.json",
    mode="0644", user="root", group="root"))

notify("clientrotate", files.template(
    name="Deploy clientrotate.service",
    src="deploy/templates/xray_server/clientrotate.service.j2",
    dest="/etc/systemd/system/clientrotate.service",
    mode="0644", user="root", group="root"))

notify("clientrotate", files.template(
    name="Deploy clientrotate.timer",
    src="deploy/templates/xray_server/clientrotate.timer.j2",
    dest="/etc/systemd/system/clientrotate.timer",
    mode="0644", user="root", group="root"))
