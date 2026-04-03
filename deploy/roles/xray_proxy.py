from pyinfra.operations import python, server, systemd
from pyinfra import host, local
from deploy.triggers import changed
from os import path

local.include(filename=path.join("deploy", "tasks", "xray_proxy", "packages.py"))
local.include(filename=path.join("deploy", "tasks", "xray_proxy", "configs.py"))

for svc in ["nftables", "nginx", "ssh", "subsrv"]:
    systemd.service(
        name=f"Enable and start {svc}",
        service=svc, running=True, enabled=True,
        restarted=changed(svc), daemon_reload=changed(svc))

server.shell(
    name="Obtain Let's Encrypt certificate",
    commands=[
        "certbot certonly --webroot -w /var/www/html"
        f" -d {host.data.proxy_domain}"
        " --non-interactive --agree-tos -m realglebivanov@gmail.com"
        " --keep-until-expiring"
        " --deploy-hook 'systemctl restart subsrv'",
    ])

systemd.service(
    name="Enable certbot renewal timer",
    service="certbot.timer", running=True, enabled=True)
