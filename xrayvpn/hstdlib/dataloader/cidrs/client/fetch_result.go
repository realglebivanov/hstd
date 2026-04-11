package client

type FetchResult struct {
	Src   Source
	CIDRs []string
	Err   error
}

func toCIDRs(results []*FetchResult) ([]string, []error) {
	var allCIDRs []string
	var errs []error
	for _, r := range results {
		if r.Err == nil {
			allCIDRs = append(allCIDRs, r.CIDRs...)
			continue
		}

		errs = append(errs, r.Err)
	}

	return dedup(allCIDRs), errs
}

func dedup(cidrs []string) []string {
	seen := make(map[string]struct{}, len(cidrs))
	out := make([]string, 0, len(cidrs))
	for _, c := range cidrs {
		if _, ok := seen[c]; !ok {
			seen[c] = struct{}{}
			out = append(out, c)
		}
	}
	return out
}
