package ruleset

import (
	"errors"
	"fmt"
	"log"
	"net"
	"syscall"

	"github.com/realglebivanov/hstd/hstdlib"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

const (
	localRuleBasePriority   = 1000
	directRuleBasePriority  = 2000
	xrayOutMarkRulePriority = 3000
)

type describedRule struct {
	rule *netlink.Rule
	desc string
}

type RuleSet []describedRule

func (rs RuleSet) SetUp() error {
	for _, r := range rs {
		if err := netlink.RuleAdd(r.rule); err != nil {
			return fmt.Errorf("set up %s rule: %w", r.desc, err)
		}
	}
	return nil
}

func (rs RuleSet) TearDown() error {
	for _, r := range rs {
		err := netlink.RuleDel(r.rule)
		if err == nil || errors.Is(err, syscall.ENOENT) {
			continue
		}
		return fmt.Errorf("tear down %s rule: %w", r.desc, err)
	}
	return nil
}

func New() (RuleSet, error) {
	apdCIDR, err := hstdlib.ParseApdCIDR()
	if err != nil {
		return nil, err
	}

	rules := []describedRule{{buildXrayOutMarkRule(), "xray out mark"}}

	for i, username := range hstdlib.DirectRouteServices {
		uid, err := hstdlib.LookupUID(username)
		if err != nil {
			log.Printf("skip %s rules: %v", username, err)
			continue
		}

		localRule := buildServiceLocalRule(apdCIDR, uid, localRuleBasePriority+i)
		directRule := buildServiceDirectRule(uid, directRuleBasePriority+i)

		rules = append(rules,
			describedRule{localRule, username + " local"},
			describedRule{directRule, username + " direct"},
		)
	}

	return rules, nil
}

func buildXrayOutMarkRule() *netlink.Rule {
	rule := netlink.NewRule()
	rule.Mark = hstdlib.XrayOutMark
	rule.Table = hstdlib.DirectRouteTable
	rule.Priority = xrayOutMarkRulePriority
	return rule
}

func buildServiceLocalRule(apdCIDR *net.IPNet, uid uint32, priority int) *netlink.Rule {
	rule := netlink.NewRule()
	rule.UIDRange = netlink.NewRuleUIDRange(uid, uid)
	rule.Dst = apdCIDR
	rule.Table = unix.RT_TABLE_MAIN
	rule.Priority = priority
	return rule
}

func buildServiceDirectRule(uid uint32, priority int) *netlink.Rule {
	rule := netlink.NewRule()
	rule.UIDRange = netlink.NewRuleUIDRange(uid, uid)
	rule.Table = hstdlib.DirectRouteTable
	rule.Priority = priority
	return rule
}
