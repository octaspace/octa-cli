//go:build linux

package vpnd

import (
	"fmt"
	"os/exec"
	"strings"
)

func ifaceUp(iface string, mtu int) error {
	if err := run("ip", "link", "set", iface, "mtu", itoa(mtu)); err != nil {
		return err
	}
	return run("ip", "link", "set", iface, "up")
}

func addAddress(iface, cidr string) error {
	return run("ip", "addr", "add", cidr, "dev", iface)
}

func addRoute(iface, cidr string) error {
	return run("ip", "route", "add", cidr, "dev", iface)
}

func addHostRoute(host, gw, iface string) error {
	return run("ip", "route", "add", host+"/32", "via", gw, "dev", iface)
}

func removeHostRoute(host string) error {
	return run("ip", "route", "del", host+"/32")
}

func defaultGateway() (gw, iface string, err error) {
	out, err := exec.Command("ip", "route", "show", "default").Output()
	if err != nil {
		return "", "", fmt.Errorf("ip route show default: %w", err)
	}
	// Format: "default via 192.168.1.1 dev eth0 ..."
	fields := strings.Fields(string(out))
	for i, f := range fields {
		if f == "via" && i+1 < len(fields) {
			gw = fields[i+1]
		}
		if f == "dev" && i+1 < len(fields) {
			iface = fields[i+1]
		}
	}
	if gw == "" || iface == "" {
		return "", "", fmt.Errorf("could not parse default gateway from: %s", string(out))
	}
	return gw, iface, nil
}

func run(name string, args ...string) error {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return wrapCmdError(name, args, err, out)
	}
	return nil
}
