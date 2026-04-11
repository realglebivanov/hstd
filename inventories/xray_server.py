from deploy import passwd, xray

hosts = [
    (xray.xray_server_addr, {
        "role": "xray_server",
        "ssh_user": "root",
        "reality_dest": f"{xray.reality_sni}:443",
        "reality_server_names": [xray.reality_sni],
        "reality_short_id": xray.reality_sid,
        "reality_private_key": passwd.reality_private_key,
        "rotate_secret": passwd.rotate_secret,
        "routing_rules": xray.routing_rules,
    }),
]