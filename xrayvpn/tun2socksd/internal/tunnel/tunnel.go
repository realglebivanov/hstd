package tunnel

import (
	"errors"
	"fmt"
	"net"

	"github.com/realglebivanov/hstd/tun2socksd/internal/tunnel/link"
	"github.com/realglebivanov/hstd/tun2socksd/internal/tunnel/nftable"
	"github.com/realglebivanov/hstd/tun2socksd/internal/tunnel/routetable"
	"github.com/realglebivanov/hstd/tun2socksd/internal/tunnel/ruleset"
	"github.com/vishvananda/netlink"
)

type Tunnel struct {
	link   *link.Link
	rules  ruleset.RuleSet
	routes *routetable.RouteTable
	nft    *nftable.NfTable
}

func New() (*Tunnel, error) {
	link, err := link.SetUp()
	if err != nil {
		return nil, err
	}

	rules, err := ruleset.New()
	if err != nil {
		return nil, fmt.Errorf("build rules: %w", err)
	}

	return &Tunnel{
		link:   link,
		rules:  rules,
		routes: routetable.New(link),
		nft:    nftable.New(),
	}, nil
}

func (tun *Tunnel) DefaultGwAddr() *net.IP {
	return &tun.link.DefaultGw.Gw
}

func (tun *Tunnel) TunAddr() *netlink.Addr {
	return tun.link.TunAddr
}

func (tun *Tunnel) SetUp() error {
	if err := tun.rules.SetUp(); err != nil {
		return fmt.Errorf("set up route rules: %w", err)
	}
	if err := tun.routes.SetUp(); err != nil {
		return fmt.Errorf("set up route table: %w", err)
	}
	if err := tun.nft.SetUp(); err != nil {
		return fmt.Errorf("set up nftables: %w", err)
	}
	return nil
}

func (tun *Tunnel) TearDown() error {
	var errs []error
	if err := tun.routes.TearDown(); err != nil {
		errs = append(errs, fmt.Errorf("tear down route table: %w", err))
	}
	if err := tun.rules.TearDown(); err != nil {
		errs = append(errs, fmt.Errorf("tear down route rules: %w", err))
	}
	if err := tun.nft.TearDown(); err != nil {
		errs = append(errs, fmt.Errorf("tear down nftables: %w", err))
	}
	return errors.Join(errs...)
}
