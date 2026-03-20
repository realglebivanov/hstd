package routing

import (
	"fmt"
	"log"

	"github.com/google/nftables"
	"github.com/google/nftables/binaryutil"
	"github.com/google/nftables/expr"
	"github.com/google/nftables/userdata"
)

const (
	nftRuleComment = "xrayvpn"
	vpnMark        = 0x00001337
)

var (
	fw4Table = &nftables.Table{
		Name:   "xray_vpn",
		Family: nftables.TableFamilyIPv4,
	}
	forwardChain = &nftables.Chain{
		Name:     "forward",
		Table:    fw4Table,
		Type:     nftables.ChainTypeFilter,
		Hooknum:  nftables.ChainHookForward,
		Priority: nftables.ChainPriorityFirst,
	}
)

func allowForwardToTun() error {
	conn, err := nftables.New()
	if err != nil {
		return fmt.Errorf("nftables conn: %w", err)
	}

	conn.AddTable(fw4Table)
	conn.AddChain(forwardChain)
	conn.AddRule(buildForwardRule("lo", TunDev))
	conn.AddRule(buildForwardRule(LanDev, TunDev))
	conn.AddRule(buildForwardRule(TunDev, WanDev))
	conn.AddRule(buildForwardRule(TunDev, LanDev))

	if err := conn.Flush(); err != nil {
		return fmt.Errorf("nft insert forward rule: %w", err)
	}

	log.Printf("nft: forward to %s allowed", TunDev)
	return nil
}

func removeForwardToTun() error {
	conn, err := nftables.New()
	if err != nil {
		return fmt.Errorf("nftables conn: %w", err)
	}
	rules, err := conn.GetRules(fw4Table, forwardChain)
	if err != nil {
		return fmt.Errorf("nft get rules: %w", err)
	}
	var count int
	for _, r := range rules {
		comment, ok := userdata.GetString(r.UserData, userdata.TypeComment)
		if !ok || comment != nftRuleComment {
			continue
		}
		if err := conn.DelRule(r); err != nil {
			return fmt.Errorf("nft del rule handle %d: %w", r.Handle, err)
		}
		count++
	}
	if count > 0 {
		if err := conn.Flush(); err != nil {
			return fmt.Errorf("nft flush deletes: %w", err)
		}
		log.Printf("nft: removed %d forward rule(s)", count)
	}
	return nil
}

func buildForwardRule(from string, to string) *nftables.Rule {
	return &nftables.Rule{
		Table: fw4Table,
		Chain: forwardChain,
		Exprs: []expr.Any{
			&expr.Meta{Key: expr.MetaKeyIIFNAME, Register: 1},
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     ifname(from),
			},
			&expr.Meta{Key: expr.MetaKeyOIFNAME, Register: 1},
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     ifname(to),
			},
			&expr.Counter{},
			&expr.Immediate{Register: 1, Data: binaryutil.NativeEndian.PutUint32(vpnMark)},
			&expr.Meta{Key: expr.MetaKeyMARK, Register: 1, SourceRegister: true},
			&expr.Verdict{Kind: expr.VerdictAccept},
		},
		UserData: userdata.AppendString(nil, userdata.TypeComment, nftRuleComment),
	}
}

func ifname(n string) []byte {
	b := make([]byte, 16)
	copy(b, n)
	return b
}
