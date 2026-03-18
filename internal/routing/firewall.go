package routing

import (
	"fmt"
	"log"

	"github.com/google/nftables"
	"github.com/google/nftables/expr"
	"github.com/google/nftables/userdata"
)

const nftRuleComment = "xray-vpn"

var (
	fw4Table     = &nftables.Table{Name: "xray_vpn", Family: nftables.TableFamilyINet}
	forwardChain = &nftables.Chain{
		Name:     "forward",
		Table:    fw4Table,
		Type:     nftables.ChainTypeFilter,
		Hooknum:  nftables.ChainHookForward,
		Priority: nftables.ChainPriorityFilter,
	}
)

func ifname(n string) []byte {
	b := make([]byte, 16)
	copy(b, n)
	return b
}

func allowForwardToTun() error {
	conn, err := nftables.New()
	if err != nil {
		return fmt.Errorf("nftables conn: %w", err)
	}
	conn.AddTable(fw4Table)
	conn.AddChain(forwardChain)
	conn.InsertRule(&nftables.Rule{
		Table: fw4Table,
		Chain: forwardChain,
		Exprs: []expr.Any{
			&expr.Meta{Key: expr.MetaKeyOIFNAME, Register: 1},
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     ifname(TunDev),
			},
			&expr.Counter{},
			&expr.Verdict{Kind: expr.VerdictAccept},
		},
		UserData: userdata.AppendString(nil, userdata.TypeComment, nftRuleComment),
	})
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
