package xrayconf

func ExpandRules(rules []RouteRule, ruCIDRs []string) []RouteRule {
	if len(ruCIDRs) == 0 {
		return rules
	}

	out := make([]RouteRule, len(rules))
	for i, r := range rules {
		out[i] = r
		if len(r.IP) > 0 {
			out[i].IP = expandIPs(r.IP, ruCIDRs)
		}
	}
	return out
}

func expandIPs(ips []string, ruCIDRs []string) []string {
	var out []string
	for _, ip := range ips {
		if ip == "cidr:ru" {
			out = append(out, ruCIDRs...)
		} else {
			out = append(out, ip)
		}
	}
	return out
}
