from deploy import passwd, xray

hosts = [
    ("192.168.2.50", {
        "role": "hstd",
        "ssh_user": "gleb",
        "_sudo": True,
        "wpa_passphrase": passwd.wpa_passphrase,
        "apd_ip": "192.168.2.50",
        "apd_cidr": "192.168.2.0/24",
        "apd_gateway_cidr": "192.168.2.50/24",
        "lan_gateway_cidr": "192.168.2.51/24",
        "dhcp_range_start": "192.168.2.100",
        "dhcp_range_end": "192.168.2.200",
        "transmission_rpc_whitelist": "127.0.0.1,192.168.2.*",
        "xray_out_mark": 31,
        "xray_traffic_mark": 4919,
        "wan_dev": "eno1",
        "apd_dev": "wlp4s0",
        "lan_dev": "enp2s0",
        "tun_dev": "xray0",
        "rotate_secret": passwd.rotate_secret,
        "xray_server_addr": xray.xray_server_addr,
        "xray_proxy_addr": xray.xray_proxy_addr,
        "reality_pbk": xray.reality_pbk,
        "reality_sni": xray.reality_sni,
        "reality_sid": xray.reality_sid,
    }),
]