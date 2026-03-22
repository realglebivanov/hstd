import subprocess

wpa_passphrase = subprocess.check_output(
    ["pass", "hstd/wpa_passphrase"],
    text=True
).strip()

hosts = [
    ("192.168.2.50", {
        "ssh_user": "gleb",
        "_sudo": True,
        "wpa_passphrase": wpa_passphrase,
        "xray_out_mark": 31,
        "xray_traffic_mark": 4919,
        "wan_dev": "eno1",
        "apd_dev": "wlp4s0",
        "lan_dev": "enp2s0",
        "tun_dev": "xray0",
    }),
]
