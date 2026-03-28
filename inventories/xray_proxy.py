from deploy import passwd, xray

hosts = [
    (xray.xray_proxy_addr, {
        "role": "xray_proxy",
        "ssh_user": "gleb",
        "_sudo": True,
        "xray_server_addr": xray.xray_server_addr,
        "rotate_secret": passwd.rotate_secret,
        "reality_pbk": xray.reality_pbk,
        "reality_sni": xray.reality_sni,
        "reality_sid": xray.reality_sid,
        "sub_path": passwd.sub_path,
        "proxy_domain": "x.hstd.space",
        "admin_user": passwd.admin_user,
        "admin_password_hash": passwd.admin_password_hash,
    }),
]
