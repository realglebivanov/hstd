package routing

import (
	"errors"
	"fmt"
	"log"
	"syscall"

	"github.com/realglebivanov/hstd/hstdlib"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

const (
	localTransmissionRulePriority = 80
	transmissionRulePriority      = 90
	xrayOutMarkRulePriority       = 100
)

func setUpRules() error {
	if err := netlink.RuleAdd(buildXrayOutMarkRule()); err != nil {
		return fmt.Errorf("set up xray out mark rule: %w", err)
	}

	uid, err := hstdlib.TransmissionUID()
	if err != nil {
		log.Printf("skip tranmission rule set up: %v", err)
		return nil
	}

	transmissionLocalRule, err := buildTransmissionLocalRule(uid)
	if err != nil {
		return fmt.Errorf("build transmission uid local rule: %w", err)
	}

	if err := netlink.RuleAdd(transmissionLocalRule); err != nil {
		return fmt.Errorf("set up transmission uid local rule: %w", err)
	}
	if err := netlink.RuleAdd(buildTransmissionRule(uid)); err != nil {
		return fmt.Errorf("set up transmission uid rule: %w", err)
	}

	return nil
}

func tearDownRules() error {
	if err := netlink.RuleDel(buildXrayOutMarkRule()); err != nil && !errors.Is(err, syscall.ENOENT) {
		return fmt.Errorf("tear down xray out mark rule: %w", err)
	}

	uid, err := hstdlib.TransmissionUID()
	if err != nil {
		log.Printf("skip tranmission rule tear down: %v", err)
		return nil
	}

	transmissionLocalRule, err := buildTransmissionLocalRule(uid)
	if err != nil {
		return fmt.Errorf("build transmission uid local rule: %w", err)
	}

	if err := netlink.RuleDel(transmissionLocalRule); err != nil && !errors.Is(err, syscall.ENOENT) {
		return fmt.Errorf("tear down transmission uid local rule: %w", err)
	}
	if err := netlink.RuleDel(buildTransmissionRule(uid)); err != nil && !errors.Is(err, syscall.ENOENT) {
		return fmt.Errorf("tear down transmission uid rule: %w", err)
	}

	return nil
}

func buildXrayOutMarkRule() *netlink.Rule {
	rule := netlink.NewRule()
	rule.Mark = hstdlib.XrayOutMark
	rule.Table = directRouteTable
	rule.Priority = xrayOutMarkRulePriority
	return rule
}

func buildTransmissionLocalRule(uid uint32) (*netlink.Rule, error) {
	apdCIDR, err := hstdlib.ParseApdCIDR()
	if err != nil {
		return nil, err
	}

	rule := netlink.NewRule()
	rule.UIDRange = netlink.NewRuleUIDRange(uid, uid)
	rule.Dst = apdCIDR
	rule.Table = unix.RT_TABLE_MAIN
	rule.Priority = localTransmissionRulePriority
	return rule, nil
}

func buildTransmissionRule(uid uint32) *netlink.Rule {
	rule := netlink.NewRule()
	rule.UIDRange = netlink.NewRuleUIDRange(uid, uid)
	rule.Table = directRouteTable
	rule.Priority = transmissionRulePriority
	return rule
}
