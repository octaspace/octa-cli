//go:build windows

package vpnd

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
)

func ifaceUp(iface string, mtu int) error {
	// Wintun interfaces are UP by default; MTU is set at tun.CreateTUN time.
	return nil
}

func addAddress(iface, cidr string) error {
	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return err
	}
	if ip.To4() != nil {
		mask := net.IP(ipNet.Mask).String()
		return run("netsh", "interface", "ip", "add", "address",
			fmt.Sprintf("name=%s", iface), ip.String(), mask)
	}
	ones, _ := ipNet.Mask.Size()
	return run("netsh", "interface", "ipv6", "add", "address",
		fmt.Sprintf("interface=%s", iface), fmt.Sprintf("%s/%d", ip, ones))
}

func addRoute(iface, cidr string) error {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return err
	}
	ones, _ := ipNet.Mask.Size()
	if ipNet.IP.To4() != nil {
		mask := net.IP(ipNet.Mask).String()
		return run("netsh", "interface", "ip", "add", "route",
			fmt.Sprintf("%s/%d", ipNet.IP, ones), iface, "0.0.0.0", mask)
	}
	return run("netsh", "interface", "ipv6", "add", "route",
		fmt.Sprintf("%s/%d", ipNet.IP, ones), iface)
}

func addHostRoute(host, gw, _ string) error {
	return run("route", "add", host, "mask", "255.255.255.255", gw)
}

func removeHostRoute(host string) error {
	return run("route", "delete", host)
}

func defaultGateway() (gw, iface string, err error) {
	out, err := exec.Command("powershell", "-NoProfile", "-Command",
		"(Get-NetRoute -DestinationPrefix '0.0.0.0/0' | Sort-Object RouteMetric | Select-Object -First 1 | Select-Object NextHop, InterfaceAlias) | ConvertTo-Csv -NoTypeInformation").Output()
	if err != nil {
		return "", "", fmt.Errorf("Get-NetRoute: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return "", "", fmt.Errorf("unexpected Get-NetRoute output")
	}
	fields := strings.Split(strings.Trim(lines[1], "\r"), ",")
	if len(fields) < 2 {
		return "", "", fmt.Errorf("unexpected Get-NetRoute fields")
	}
	gw = strings.Trim(fields[0], `"`)
	iface = strings.Trim(fields[1], `"`)
	if gw == "" {
		return "", "", fmt.Errorf("empty gateway from Get-NetRoute")
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
