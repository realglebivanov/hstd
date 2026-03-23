package nftable

import (
	"fmt"
	"log"

	"github.com/google/nftables"
	"github.com/google/nftables/binaryutil"
	"github.com/google/nftables/expr"
	"github.com/realglebivanov/hstd/hstdlib"
)

type NfTable struct {
	table *nftables.Table
	chain *nftables.Chain
	rules []*nftables.Rule
}

func New() *NfTable {
	table := &nftables.Table{
		Name:   "xray_vpn",
		Family: nftables.TableFamilyIPv4,
	}
	chain := &nftables.Chain{
		Name:     "forward",
		Table:    table,
		Type:     nftables.ChainTypeFilter,
		Hooknum:  nftables.ChainHookForward,
		Priority: nftables.ChainPriorityFirst,
	}

	nft := NfTable{table: table, chain: chain}
	nft.forward("lo", hstdlib.TunDev)
	nft.forward(hstdlib.ApdDev, hstdlib.TunDev)
	nft.forward(hstdlib.TunDev, hstdlib.WanDev)
	nft.forward(hstdlib.TunDev, hstdlib.ApdDev)

	return &nft
}

func (nft *NfTable) SetUp() error {
	conn, err := nftables.New()
	if err != nil {
		return fmt.Errorf("nftables conn: %w", err)
	}

	conn.AddTable(nft.table)
	conn.AddChain(nft.chain)
	for _, r := range nft.rules {
		conn.AddRule(r)
	}

	if err := conn.Flush(); err != nil {
		return fmt.Errorf("nft insert forward rules: %w", err)
	}

	log.Printf("nft: forward to %s allowed", hstdlib.TunDev)
	return nil
}

func (nft *NfTable) TearDown() error {
	conn, err := nftables.New()
	if err != nil {
		return fmt.Errorf("nftables conn: %w", err)
	}

	conn.DelTable(nft.table)

	if err := conn.Flush(); err != nil {
		return fmt.Errorf("nft delete table: %w", err)
	}
	log.Printf("nft: table %s removed", nft.table.Name)
	return nil
}

func (nft *NfTable) forward(from string, to string) {
	nft.rules = append(nft.rules, &nftables.Rule{
		Table: nft.table,
		Chain: nft.chain,
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
			&expr.Immediate{Register: 1, Data: binaryutil.NativeEndian.PutUint32(hstdlib.XrayTrafficMark)},
			&expr.Meta{Key: expr.MetaKeyMARK, Register: 1, SourceRegister: true},
			&expr.Verdict{Kind: expr.VerdictAccept},
		},
	})
}

func ifname(n string) []byte {
	b := make([]byte, 16)
	copy(b, n)
	return b
}
