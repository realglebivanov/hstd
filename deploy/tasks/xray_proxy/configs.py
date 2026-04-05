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
    src="deploy/templates/xray_proxy/journald.conf",
    dest="/etc/systemd/journald.conf.d/retention.conf",
    mode="0644", user="root", group="root"))

files.directory(
    name="Ensure /var/www/html exists",
    path="/var/www/html",
    present=True, mode="0755", user="www-data", group="www-data")

notify("nftables", files.template(
    name="Deploy /etc/nftables.conf",
    src="deploy/templates/xray_proxy/nftables.conf.j2",
    dest="/etc/nftables.conf",
    mode="0644", user="root", group="root"))

notify("ssh", files.template(
    name="Deploy /etc/ssh/sshd_config",
    src="deploy/templates/xray_proxy/sshd_config.j2",
    dest="/etc/ssh/sshd_config",
    mode="0644", user="root", group="root"))

notify("nginx", files.template(
    name="Deploy nginx.conf with stream proxy",
    src="deploy/templates/xray_proxy/nginx.conf.j2",
    dest="/etc/nginx/nginx.conf",
    mode="0644", user="root", group="root"))

notify("nginx", files.template(
    name="Deploy nginx default site",
    src="deploy/templates/xray_proxy/nginx-default.conf.j2",
    dest="/etc/nginx/sites-available/default",
    mode="0644", user="root", group="root"))

files.template(
    name="Deploy sysctl config",
    src="deploy/templates/xray_proxy/sysctl.conf.j2",
    dest="/etc/sysctl.d/99-xray-proxy.conf",
    mode="0644", user="root", group="root")

server.shell(
    name="Apply sysctl tuning",
    commands=["sysctl -p /etc/sysctl.d/99-xray-proxy.conf"])

files.directory(
    name="Ensure /etc/subsrv exists",
    path="/etc/subsrv",
    present=True, mode="0755", user="root", group="root")

notify("subsrv", files.put(
    name="Deploy subsrv config",
    src=StringIO(json.dumps(host.data.subsrv_config, ensure_ascii=False)),
    dest="/etc/subsrv/config.json",
    mode="0644", user="root", group="root"))

notify("subsrv", files.template(
    name="Deploy subsrv.service",
    src="deploy/templates/xray_proxy/subsrv.service.j2",
    dest="/etc/systemd/system/subsrv.service",
    mode="0644", user="root", group="root"))
