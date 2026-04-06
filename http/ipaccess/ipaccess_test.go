package ipaccess

import (
	"net"
	"testing"
)

func TestIPAccess_ParseList(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantIPs  int
		wantNets int
	}{
		{"empty string", "", 0, 0},
		{"whitespace only", "   ", 0, 0},
		{"single IP", "192.168.1.1", 1, 0},
		{"single CIDR", "10.0.0.0/8", 0, 1},
		{"mixed IP and CIDR", "192.168.1.1,10.0.0.0/8", 1, 1},
		{"multiple IPs", "1.2.3.4,5.6.7.8", 2, 0},
		{"multiple CIDRs", "10.0.0.0/8,172.16.0.0/12", 0, 2},
		{"invalid entry skipped", "not-an-ip,192.168.1.1", 1, 0},
		{"invalid CIDR skipped", "999.0.0.0/8,192.168.1.1", 1, 0},
		{"empty parts ignored", "192.168.1.1,,10.0.0.0/8", 1, 1},
		{"whitespace trimmed", " 192.168.1.1 , 10.0.0.0/8 ", 1, 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ips, nets := ParseList(tc.input)
			if len(ips) != tc.wantIPs {
				t.Errorf("ParseList(%q) IPs = %d, want %d", tc.input, len(ips), tc.wantIPs)
			}
			if len(nets) != tc.wantNets {
				t.Errorf("ParseList(%q) Nets = %d, want %d", tc.input, len(nets), tc.wantNets)
			}
		})
	}
}

func TestIPAccess_Evaluate(t *testing.T) {
	ip1 := net.ParseIP("192.168.1.100")
	ip2 := net.ParseIP("10.0.0.5")
	ip3 := net.ParseIP("172.16.0.50")

	_, net1, _ := net.ParseCIDR("10.0.0.0/8")
	_, net2, _ := net.ParseCIDR("172.16.0.0/12")

	tests := []struct {
		name  string
		ip    net.IP
		lists *Lists
		want  bool
	}{
		{
			name:  "nil lists allows all",
			ip:    ip1,
			lists: nil,
			want:  true,
		},
		{
			name: "empty lists allows all",
			ip:   ip1,
			lists: &Lists{},
			want: true,
		},
		{
			name: "whitelist IP - IP in whitelist",
			ip:   ip1,
			lists: &Lists{
				WhitelistIPs: []net.IP{ip1},
			},
			want: true,
		},
		{
			name: "whitelist IP - IP not in whitelist",
			ip:   ip2,
			lists: &Lists{
				WhitelistIPs: []net.IP{ip1},
			},
			want: false,
		},
		{
			name: "whitelist CIDR - IP in whitelist net",
			ip:   ip2,
			lists: &Lists{
				WhitelistNets: []*net.IPNet{net1},
			},
			want: true,
		},
		{
			name: "whitelist CIDR - IP not in whitelist net",
			ip:   ip1,
			lists: &Lists{
				WhitelistNets: []*net.IPNet{net1},
			},
			want: false,
		},
		{
			name: "blacklist IP - IP in blacklist",
			ip:   ip1,
			lists: &Lists{
				BlacklistIPs: []net.IP{ip1},
			},
			want: false,
		},
		{
			name: "blacklist IP - IP not in blacklist",
			ip:   ip2,
			lists: &Lists{
				BlacklistIPs: []net.IP{ip1},
			},
			want: true,
		},
		{
			name: "blacklist CIDR - IP in blacklist net",
			ip:   ip3,
			lists: &Lists{
				BlacklistNets: []*net.IPNet{net2},
			},
			want: false,
		},
		{
			name: "blacklist CIDR - IP not in blacklist net",
			ip:   ip1,
			lists: &Lists{
				BlacklistNets: []*net.IPNet{net2},
			},
			want: true,
		},
		{
			name: "whitelist takes precedence - IP in whitelist",
			ip:   ip1,
			lists: &Lists{
				WhitelistIPs: []net.IP{ip1},
				BlacklistIPs: []net.IP{ip1},
			},
			want: true,
		},
		{
			name: "whitelist present - IP not in whitelist (even if not blacklisted)",
			ip:   ip2,
			lists: &Lists{
				WhitelistIPs: []net.IP{ip1},
			},
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Evaluate(tc.ip, tc.lists)
			if got != tc.want {
				t.Errorf("Evaluate(%v, lists) = %v, want %v", tc.ip, got, tc.want)
			}
		})
	}
}

func TestIPAccess_ParseIPList(t *testing.T) {
	// Original stub test name reused - tests ParseList via single-IP case
	ips, nets := ParseList("192.0.2.1")
	if len(ips) != 1 || len(nets) != 0 {
		t.Errorf("expected 1 IP, 0 nets; got %d IPs, %d nets", len(ips), len(nets))
	}
}

func TestIPAccess_ParseLists(t *testing.T) {
	lists := ParseLists("192.168.1.1,10.0.0.0/8", "1.2.3.4")
	if len(lists.WhitelistIPs) != 1 {
		t.Errorf("expected 1 whitelist IP, got %d", len(lists.WhitelistIPs))
	}
	if len(lists.WhitelistNets) != 1 {
		t.Errorf("expected 1 whitelist net, got %d", len(lists.WhitelistNets))
	}
	if len(lists.BlacklistIPs) != 1 {
		t.Errorf("expected 1 blacklist IP, got %d", len(lists.BlacklistIPs))
	}
	if len(lists.BlacklistNets) != 0 {
		t.Errorf("expected 0 blacklist nets, got %d", len(lists.BlacklistNets))
	}
}

func TestIPAccess_ParseLists_Empty(t *testing.T) {
	lists := ParseLists("", "")
	if lists == nil {
		t.Fatal("ParseLists returned nil")
	}
	if len(lists.WhitelistIPs) != 0 || len(lists.WhitelistNets) != 0 ||
		len(lists.BlacklistIPs) != 0 || len(lists.BlacklistNets) != 0 {
		t.Error("expected all empty lists")
	}
}
