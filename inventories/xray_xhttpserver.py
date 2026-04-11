from deploy import passwd, xray

hosts = [
    (xray.xray_xhttpserver_addr, {
        "role": "xray_xhttpserver",
        "ssh_user": "root",
        "rotate_secret": passwd.rotate_secret,
        "routing_rules": xray.routing_rules,
        "xhttp_source_domain": xray.xhttp_source_domain,
        "xhttp_path": xray.xhttp_path,
        "letsencrypt_email": xray.letsencrypt_email,
    }),
]
