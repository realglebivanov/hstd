from deploy import passwd, xray

hosts = [
    (xray.xray_proxy_addr, {
        "role": "xray_proxy",
        "ssh_user": "gleb",
        "_sudo": True,
        "xray_server_addr": xray.xray_server_addr,
        "rotate_secret": passwd.rotate_secret,
        "sub_path": passwd.sub_path,
        "proxy_domain": "x.hstd.space",
        "admin_user": passwd.admin_user,
        "admin_password_hash": passwd.admin_password_hash,
        "subsrv_config": {
            "servers": [
                {
                    "remark": "Обычный ВПН",
                    "host": xray.xray_server_addr,
                    "realityPbk": xray.reality_pbk,
                    "realitySni": xray.reality_sni,
                    "realitySid": xray.reality_sid,
                },
                {
                    "remark": "Обход белых списков(tcp, reality)",
                    "host": xray.xray_proxy_addr,
                    "realityPbk": xray.reality_pbk,
                    "realitySni": xray.reality_sni,
                    "realitySid": xray.reality_sid,
                },
                {
                    "remark": "Обход белых списков(xhttp, tls)",
                    "host": xray.xhttp_cdn_domain,
                    "xhttpPath": xray.xhttp_path,
                },
            ],
            "routingRules": [
                {"type": "field", "outboundTag": "proxy", "domain": ["domain:yonote.ru"]},
                {"type": "field", "outboundTag": "direct", "ip": ["geoip:ru", "geoip:private"]},
                {"type": "field", "outboundTag": "direct", "domain": ["geosite:category-ru", "geosite:category-gov-ru"]},
                {"type": "field", "outboundTag": "proxy", "network": "tcp,udp"},
            ],
        },
    }),
]
