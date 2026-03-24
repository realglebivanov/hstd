import subprocess

def _pass(key):
    return subprocess.check_output(["pass", key], text=True).strip()

rotate_secret = _pass("hstd/rotate_secret")
wpa_passphrase = _pass("hstd/wpa_passphrase")
sub_path = _pass("hstd/xray_proxy/sub_path")
reality_private_key = _pass("hstd/xray_server/reality_private_key")
