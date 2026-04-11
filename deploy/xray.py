import re, os

def _xray_version_from_gomod():
    gomod = os.path.join(os.path.dirname(__file__), "..", "xrayvpn", "xrayvpnd", "go.mod")
    with open(gomod) as f:
        for line in f:
            m = re.search(r"github\.com/xtls/xray-core v1\.(\d{2})(\d{2})(\d{2})\.\d+", line)
            if m:
                return f"{int(m.group(1))}.{int(m.group(2))}.{int(m.group(3))}"
    raise RuntimeError("xray-core version not found in xrayvpnd/go.mod")

xray_version = _xray_version_from_gomod()

xray_xhttpserver_addr = "5.252.21.248"
xray_server_addr = "80.71.157.96"
xray_proxy_addr = "158.160.19.75"

reality_pbk = "-wQcqdK1CZB9rcW3zeM3W2qx5lDENo9g3YN-jSU-LWI"
reality_sni = "yandex.ru"
reality_sid = "e2174ad2204ca5c5"

proxy_domain = "x.hstd.space"
xhttp_source_domain = "cdn.hstd.space"
xhttp_cdn_domain = "pub.cdn.hstd.space"
xhttp_path = "/images/625385d043bfac1b"

letsencrypt_email = "realglebivanov@gmail.com"
