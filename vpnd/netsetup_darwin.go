//go:build darwin

package vpnd

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
)

func ifaceUp(iface string, mtu int) error {
	if err := run("ifconfig", iface, "mtu", itoa(mtu)); err != nil {
		return err
	}
	return run("ifconfig", iface, "up")
}

func addAddress(iface, cidr string) error {
	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return err
	}
	if ip.To4() != nil {
		ones, _ := ipNet.Mask.Size()
		if ones == 32 {
			return run("ifconfig", iface, "inet", ip.String(), ip.String())
		}
		return run("ifconfig", iface, "inet", ip.String(), "netmask", dotted(ipNet.Mask))
	}
	ones, _ := ipNet.Mask.Size()
	return run("ifconfig", iface, "inet6", fmt.Sprintf("%s/%d", ip, ones))
}

func addRoute(iface, cidr string) error {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return err
	}
	if ipNet.IP.To4() != nil {
		return run("route", "add", "-net", ipNet.String(), "-interface", iface)
	}
	return run("route", "add", "-inet6", ipNet.String(), "-interface", iface)
}

func addHostRoute(host, gw, _ string) error {
	return run("route", "add", "-host", host, gw)
}

func removeHostRoute(host string) error {
	return run("route", "delete", "-host", host)
}

func defaultGateway() (gw, iface string, err error) {
	out, err := exec.Command("route", "-n", "get", "default").Output()
	if err != nil {
		return "", "", fmt.Errorf("route -n get default: %w", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "gateway:") {
			gw = strings.TrimSpace(strings.TrimPrefix(line, "gateway:"))
		}
		if strings.HasPrefix(line, "interface:") {
			iface = strings.TrimSpace(strings.TrimPrefix(line, "interface:"))
		}
	}
	if gw == "" || iface == "" {
		return "", "", fmt.Errorf("could not parse default gateway from route output")
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

func dotted(mask net.IPMask) string {
	return fmt.Sprintf("%d.%d.%d.%d", mask[0], mask[1], mask[2], mask[3])
}
