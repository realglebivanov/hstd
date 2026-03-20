from pyinfra.operations import systemd, server
from operations.triggers import changed

import operations.packages
import operations.configs

server.user(
    name="Configure gleb user with SSH key",
    user="gleb",
    groups=["xrayvpn"],
    append=True,
    public_keys=[
        "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQChx5QLwUAa7LWQAFai5sGVFKVlFtSO8iEI/7Y3Vonbf/FGz14N6wk71VOK4k+aa9Pr30EMhAqK8mlPmLVIrWZgmxYmTqXNds81VbCWu0HZvql7FyBbCoLKg+HBt9vYiD1AYhLwMG7bMrc/5uXynFJuB+GkbxHNIvpREfe1445JX6xMksDvHnVkelkbbD20+xukOpK8jXBTPdxepsN6mGYb7M+KbK7PdjHawhnTgt/DDPVhyEvxBOcHJB6iNok1Q27OIFtsEjEEI0bSAvQKY3PPBYdbnYqF4PBHA6kYGnQyyMMYQ7jCqX80GyYbHjXBZ3B8SW1ge6L2Q034ZJvgUnObsgiomBU87KA9chG1Aob4yt8KE6sS69UltdoycsRIK5dASA4prHl6/yiG126Fz3EkMdSv5+xjpdLmHlwiXbirGiQ4XP83dbySIhdg7nxoif5oski+/+4pzsnZNCZuXhRN4Qx4jP6JVnCwaR0j0LPk4BEuYj2xxyJYHh2XL7N/Qn8= gleb@local",
    ],
)

for svc in ["nftables", "dnsmasq", "hostapd", "xrayvpnd", "navidrome", "sshd", "networking"]:
    systemd.service(
        name=f"Enable and start {svc}",
        service=svc,
        running=True,
        enabled=True,
        restarted=changed(svc),
        daemon_reload=changed(svc),
    )
