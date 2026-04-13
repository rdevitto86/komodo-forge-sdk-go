package ipaccess

import (
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/rdevitto86/komodo-forge-sdk-go/config"
	httpErr "github.com/rdevitto86/komodo-forge-sdk-go/http/errors"
	httpReq "github.com/rdevitto86/komodo-forge-sdk-go/http/request"
	logger "github.com/rdevitto86/komodo-forge-sdk-go/logging/runtime"
)

var (
	ipOnce sync.Once
	lists Lists
)

// Enforces allow/deny rules based on client IP.
func IPAccessMiddleware(next http.Handler) http.Handler {
	// lazy-parse env config once
	ipOnce.Do(func() {
		wlIPs, wlNets := ParseList(config.GetConfigValue("IP_WHITELIST"))
		blIPs, blNets := ParseList(config.GetConfigValue("IP_BLACKLIST"))
		logger.Debug("parsed IP whitelist: ", logger.Attr("whitelist", wlIPs))
		logger.Debug("parsed IP blacklist: ", logger.Attr("blacklist", blIPs))
	
		lists = Lists{
			WhitelistIPs: wlIPs,
			WhitelistNets: wlNets,
			BlacklistIPs: blIPs,
			BlacklistNets: blNets,
		}
	})

	return http.HandlerFunc(func(wtr http.ResponseWriter, req *http.Request) {
		client := httpReq.GetClientKey(req)
		if client == "" {
			logger.Error("unable to determine client IP", fmt.Errorf("unable to determine client IP"))
			httpErr.SendError(
				wtr, req, httpErr.Global.Forbidden, httpErr.WithDetail("unable to determine client IP"),
			)
			return
		}

		ip := net.ParseIP(client)
		if ip == nil {
			// Try to trim potential port if present
			host, _, err := net.SplitHostPort(client)
			if err == nil {
				ip = net.ParseIP(host)
			}
		}
		if ip == nil {
			logger.Error("invalid client IP: " + client, fmt.Errorf("invalid client IP"))
			httpErr.SendError(
				wtr, req, httpErr.Global.Forbidden, httpErr.WithDetail("invalid client IP"),
			)
			return
		}

		allowed := Evaluate(ip, &lists)
		if !allowed {
			logger.Error("access denied for client ip: " + client, fmt.Errorf("access denied for client IP"))
			httpErr.SendError(
				wtr, req, httpErr.Global.Forbidden, httpErr.WithDetail("access denied for client IP"),
			)
			return
		}

		next.ServeHTTP(wtr, req)
	})
}
