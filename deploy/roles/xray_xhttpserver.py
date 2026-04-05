from pyinfra.operations import server, systemd
from pyinfra import host, local
from deploy.triggers import changed
from os import path

local.include(filename=path.join("deploy", "tasks", "xray_xhttpserver", "packages.py"))
local.include(filename=path.join("deploy", "tasks", "xray_xhttpserver", "configs.py"))

for svc in ["nftables", "nginx", "xray", "ssh", "clientrotate"]:
    systemd.service(
        name=f"Enable and start {svc}",
        service=svc, running=True, enabled=True,
        restarted=changed(svc), daemon_reload=changed(svc))

systemd.service(
    name="Restart systemd-journald",
    service="systemd-journald",
    restarted=changed("systemd-journald"))

systemd.service(
    name="Enable and start clientrotate.timer",
    service="clientrotate.timer", running=True, enabled=True,
    restarted=changed("clientrotate"), daemon_reload=changed("clientrotate"))

server.shell(
    name="Obtain Let's Encrypt certificate",
    commands=[
        "certbot certonly --webroot -w /var/www/html"
        f" -d {host.data.xhttp_source_domain}"
        f" --non-interactive --agree-tos -m {host.data.letsencrypt_email}"
        " --keep-until-expiring"
        " --deploy-hook 'chgrp -R xray-cert /etc/letsencrypt/archive"
        " && find /etc/letsencrypt/archive -name privkey\\* -exec chmod g+r {} +"
        " && systemctl restart xray'",
    ])

systemd.service(
    name="Enable certbot renewal timer",
    service="certbot.timer", running=True, enabled=True)

server.shell(
    name="Run initial client rotation",
    commands=["systemctl start clientrotate.service"])
