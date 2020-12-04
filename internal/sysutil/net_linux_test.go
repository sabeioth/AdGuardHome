//+build linux

package sysutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDHCPCDStaticConfig(t *testing.T) {
	testCases := []struct {
		name string
		data []byte
		want bool
	}{{
		name: "has_not",
		data: []byte("#comment\n# comment\n\ninterface eth0\nstatic ip_address=192.168.0.1/24\n\n# interface wlan0\nstatic ip_address=192.168.1.1/24\n\n# comment\n"),
		want: false,
	}, {
		name: "has",
		data: []byte("#comment\n# comment\n\ninterface eth0\nstatic ip_address=192.168.0.1/24\n\n# interface wlan0\nstatic ip_address=192.168.1.1/24\n\n# comment\n\ninterface wlan0\n# comment\nstatic ip_address=192.168.2.1/24\n"),
		want: true,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, dhcpcdStaticConfig(tc.data, "wlan0"))
		})
	}
}

func TestIfacesStaticConfig(t *testing.T) {
	testCases := []struct {
		name string
		data []byte
		want bool
	}{{
		name: "has_not",
		data: []byte("allow-hotplug enp0s3\n\n#iface enp0s3 inet static\n#  address 192.168.0.200\n#  netmask 255.255.255.0\n#  gateway 192.168.0.1\n\niface enp0s3 inet dhcp\n"),
		want: false,
	}, {
		name: "has",
		data: []byte("allow-hotplug enp0s3\n\niface enp0s3 inet static\n  address 192.168.0.200\n  netmask 255.255.255.0\n  gateway 192.168.0.1\n\n#iface enp0s3 inet dhcp\n"),
		want: true,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ifacesStaticConfig(tc.data, "enp0s3"))
		})
	}
}

func TestSetStaticIPdhcpcdConf(t *testing.T) {
	dhcpcdConf := `
interface wlan0
static ip_address=192.168.0.2/24
static routers=192.168.0.1
static domain_name_servers=192.168.0.2

`
	s := updateStaticIPdhcpcdConf("wlan0", "192.168.0.2/24", "192.168.0.1", "192.168.0.2")
	assert.Equal(t, dhcpcdConf, s)

	// without gateway
	dhcpcdConf = `
interface wlan0
static ip_address=192.168.0.2/24
static domain_name_servers=192.168.0.2

`
	s = updateStaticIPdhcpcdConf("wlan0", "192.168.0.2/24", "", "192.168.0.2")
	assert.Equal(t, dhcpcdConf, s)
}
