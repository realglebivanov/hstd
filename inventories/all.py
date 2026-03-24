from inventories.hstd import hosts as hstd_hosts
from inventories.xray_server import hosts as xray_server_hosts
from inventories.xray_proxy import hosts as xray_proxy_hosts

hosts = hstd_hosts + xray_server_hosts + xray_proxy_hosts
