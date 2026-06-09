package ipaccess

import (
	"fmt"
	"net"
	"strings"

	logger "github.com/rdevitto86/komodo-forge-sdk-go/logging/runtime"
)

type Lists struct {
	WhitelistIPs  []net.IP
	WhitelistNets []*net.IPNet
	BlacklistIPs  []net.IP
	BlacklistNets []*net.IPNet
}

// Returns true if the ip is allowed according to the provided lists.
func Evaluate(ip net.IP, lists *Lists) bool {
	if lists == nil {
		return true
	}

	// If whitelist present, only allow those entries
	if len(lists.WhitelistIPs) > 0 || len(lists.WhitelistNets) > 0 {
		return ipInList(ip, lists.WhitelistIPs, lists.WhitelistNets)
	}

	// If blacklisted, deny
	if ipInList(ip, lists.BlacklistIPs, lists.BlacklistNets) {
		return false
	}
	return true
}

func ipInList(ip net.IP, ips []net.IP, nets []*net.IPNet) bool {
	for _, a := range ips {
		if a.Equal(ip) {
			return true
		}
	}
	for _, n := range nets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// Parses a comma-separated list of IPs or CIDR ranges into separate slices, returning an
// error that names every entry it could not parse so a malformed allowlist/denylist fails
// loudly at config time instead of silently dropping rules (which can fail open).
func ParseListStrict(raw string) (ips []net.IP, nets []*net.IPNet, err error) {
	var invalid []string
	for p := range strings.SplitSeq(raw, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}

		if strings.Contains(p, "/") {
			if _, network, e := net.ParseCIDR(p); e == nil {
				nets = append(nets, network)
				continue
			}
		}
		if ip := net.ParseIP(p); ip != nil {
			ips = append(ips, ip)
			continue
		}
		invalid = append(invalid, p)
	}
	if len(invalid) > 0 {
		return ips, nets, fmt.Errorf("ignored %d invalid entries: %s", len(invalid), strings.Join(invalid, ", "))
	}
	return ips, nets, nil
}

// Parses a comma-separated list of IPs or CIDR ranges into separate slices; invalid entries
// are silently ignored. Prefer ParseListStrict when a malformed entry must not pass unnoticed.
func ParseList(raw string) (ips []net.IP, nets []*net.IPNet) {
	ips, nets, _ = ParseListStrict(raw)
	return ips, nets
}

// Parses both whitelist and blacklist raw strings and returns a fully populated Lists struct,
// logging a warning for any invalid entries rather than dropping them silently.
func ParseLists(whitelistRaw, blacklistRaw string) *Lists {
	wlIPs, wlNets, wlErr := ParseListStrict(whitelistRaw)
	if wlErr != nil {
		logger.Warn("ip whitelist has invalid entries", "error", wlErr.Error())
	}
	blIPs, blNets, blErr := ParseListStrict(blacklistRaw)
	if blErr != nil {
		logger.Warn("ip blacklist has invalid entries", "error", blErr.Error())
	}
	return &Lists{WhitelistIPs: wlIPs, WhitelistNets: wlNets, BlacklistIPs: blIPs, BlacklistNets: blNets}
}
