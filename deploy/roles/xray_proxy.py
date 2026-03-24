from pyinfra.operations import python, systemd
from pyinfra import host, local
from deploy.triggers import changed
from os import path

local.include(filename=path.join("tasks", "xray_proxy", "packages.py"))
local.include(filename=path.join("tasks", "xray_proxy", "configs.py"))

for svc in ["nftables", "nginx", "ssh", "subsrv"]:
    systemd.service(
        name=f"Enable and start {svc}",
        service=svc, running=True, enabled=True,
        restarted=changed(svc), daemon_reload=changed(svc))

python.call(
    name="Subscription URL",
    function=lambda: print(f"\n  http://{host.name}:8080/{host.data.sub_path}\n"))
