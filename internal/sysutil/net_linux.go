//+build linux

package sysutil

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"

	"github.com/AdguardTeam/AdGuardHome/internal/aghio"
	"github.com/AdguardTeam/AdGuardHome/internal/util"
	"github.com/AdguardTeam/golibs/file"
)

// maxConfigFileSize is the maximum length of interfaces configuration file.
const maxConfigFileSize = 1024 * 1024

func ifaceHasStaticIP(ifaceName string) (has bool, err error) {
	var f *os.File
	for _, check := range []struct {
		checker  func([]byte, string) bool
		filePath string
	}{{
		checker:  dhcpcdStaticConfig,
		filePath: "/etc/dhcpcd.conf",
	}, {
		checker:  ifacesStaticConfig,
		filePath: "/etc/network/interfaces",
	}} {
		f, err = os.Open(check.filePath)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return false, err
		}
		defer f.Close()

		fileReadCloser, err := aghio.LimitReadCloser(f, maxConfigFileSize)
		if err != nil {
			return false, err
		}
		defer fileReadCloser.Close()

		b, err := ioutil.ReadAll(fileReadCloser)
		if err != nil {
			return false, err
		}

		has = check.checker(b, ifaceName)
		if has {
			break
		}
	}

	return has, nil
}

// dhcpcdStaticConfig checks if interface is configured by /etc/dhcpcd.conf to
// have a static IP.
func dhcpcdStaticConfig(b []byte, ifaceName string) (has bool) {
	lines := strings.Split(string(b), "\n")
	nameLine := fmt.Sprintf("interface %s", ifaceName)
	withinInterfaceCtx := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if withinInterfaceCtx && len(line) == 0 {
			// an empty line resets our state
			withinInterfaceCtx = false
		}

		if len(line) == 0 || line[0] == '#' {
			continue
		}

		if !withinInterfaceCtx {
			if line == nameLine {
				// we found our interface
				withinInterfaceCtx = true
			}
		} else {
			if strings.HasPrefix(line, "interface ") {
				// we found another interface - reset our state
				withinInterfaceCtx = false
				continue
			}
			if strings.HasPrefix(line, "static ip_address=") {
				return true
			}
		}
	}
	return false
}

// ifacesStaticConfig checks if interface is configured by
// /etc/network/interfaces to have a static IP.
func ifacesStaticConfig(b []byte, ifaceName string) (has bool) {
	lines := strings.Split(string(b), "\n")
	ifacePrefix := fmt.Sprintf("iface %s ", ifaceName)

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if len(line) == 0 || line[0] == '#' {
			continue
		}

		if strings.HasPrefix(line, ifacePrefix) && strings.HasSuffix(line, "static") {
			return true
		}
	}

	return false
}

func ifaceSetStaticIP(ifaceName string) (err error) {
	ip := util.GetSubnet(ifaceName)
	if len(ip) == 0 {
		return errors.New("can't get IP address")
	}

	ip4, _, err := net.ParseCIDR(ip)
	if err != nil {
		return err
	}
	gatewayIP := GatewayIP(ifaceName)
	add := updateStaticIPdhcpcdConf(ifaceName, ip, gatewayIP, ip4.String())

	body, err := ioutil.ReadFile("/etc/dhcpcd.conf")
	if err != nil {
		return err
	}

	body = append(body, []byte(add)...)
	err = file.SafeWrite("/etc/dhcpcd.conf", body)
	if err != nil {
		return err
	}

	return nil
}

// updateStaticIPdhcpcdConf sets static IP address for the interface by writing
// into dhcpd.conf.
func updateStaticIPdhcpcdConf(ifaceName, ip, gatewayIP, dnsIP string) string {
	var body []byte

	add := fmt.Sprintf("\ninterface %s\nstatic ip_address=%s\n",
		ifaceName, ip)
	body = append(body, []byte(add)...)

	if len(gatewayIP) != 0 {
		add = fmt.Sprintf("static routers=%s\n",
			gatewayIP)
		body = append(body, []byte(add)...)
	}

	add = fmt.Sprintf("static domain_name_servers=%s\n\n",
		dnsIP)
	body = append(body, []byte(add)...)

	return string(body)
}
