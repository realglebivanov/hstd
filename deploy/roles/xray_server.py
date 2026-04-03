from pyinfra.operations import server, systemd
from pyinfra import host, local
from deploy.triggers import changed
from os import path

local.include(filename=path.join("deploy", "tasks", "xray_server", "packages.py"))
local.include(filename=path.join("deploy", "tasks", "xray_server", "configs.py"))

for svc in ["nftables", "nginx", "xray", "ssh", "clientrotate"]:
    systemd.service(
        name=f"Enable and start {svc}",
        service=svc, running=True, enabled=True,
        restarted=changed(svc), daemon_reload=changed(svc))

systemd.service(
    name="Enable and start clientrotate.timer",
    service="clientrotate.timer", running=True, enabled=True,
    restarted=changed("clientrotate"), daemon_reload=changed("clientrotate"))

server.shell(
    name="Run initial client rotation",
    commands=["systemctl start clientrotate.service"])
