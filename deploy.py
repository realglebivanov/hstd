from pyinfra import host, local
from os import path

role = host.data.get("role")
if role:
    local.include(filename=path.join("deploy", "roles", f"{role}.py"))
else:
    raise ValueError(f"Host {host.name} has no 'role' defined in inventory")
