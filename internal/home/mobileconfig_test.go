package home

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"howett.net/plist"
)

func TestHostFromHostport(t *testing.T) {
	testCases := []struct {
		name string
		host string
		want string
		err  error
	}{{
		name: "basic_hostname",
		host: "example.com",
		want: "example.com",
		err:  nil,
	}, {
		name: "basic_hostname_port",
		host: "example.com:80",
		want: "example.com",
		err:  nil,
	}, {
		name: "basic_ipv4",
		host: "1.2.3.4",
		want: "1.2.3.4",
		err:  nil,
	}, {
		name: "basic_ipv4_port",
		host: "1.2.3.4:80",
		want: "1.2.3.4",
		err:  nil,
	}, {
		name: "basic_ipv6",
		host: "2001:db8::68",
		want: "2001:db8::68",
		err:  nil,
	}, {
		name: "closed_ipv6",
		host: "[2001:db8::68]",
		want: "",
		err:  &net.AddrError{Err: "missing port in address", Addr: "[2001:db8::68]"},
	}, {
		name: "good_ipv6_port",
		host: "[1:2:3::4]:80",
		want: "1:2:3::4",
		err:  nil,
	}, {
		name: "bad_ipv6_port",
		host: "[1:2:3::4] :80",
		want: "",
		err:  &net.AddrError{Err: "missing port in address", Addr: "[1:2:3::4] :80"},
	}, {
		name: "spaced_ipv6_port",
		host: "[ 1:2:3::4]:80",
		want: " 1:2:3::4",
		err:  nil,
	}, {
		name: "prespaced_ipv6_port",
		host: " [1:2:3::4]:80",
		want: "1:2:3::4",
		err:  nil,
	}, {
		name: "bad_ipv4",
		host: "123.123.123.123::80",
		want: "123.123.123.123::80",
		err:  nil,
	}, {
		name: "bad_brackets",
		host: "[1:2:3::4]]:80",
		want: "",
		err:  &net.AddrError{Err: "missing port in address", Addr: "[1:2:3::4]]:80"},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			h, err := hostFromHostport(tc.host)
			assert.Equal(t, tc.want, h)
			assert.Equal(t, tc.err, err)
		})
	}
}

func TestHandleMobileConfigDot(t *testing.T) {
	var err error

	var r *http.Request
	r, err = http.NewRequest(http.MethodGet, "https://example.com:12345/apple/dot.mobileconfig", nil)
	assert.Nil(t, err)

	w := httptest.NewRecorder()

	handleMobileConfigDot(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	var mc MobileConfig
	_, err = plist.Unmarshal(w.Body.Bytes(), &mc)
	assert.Nil(t, err)

	if assert.Equal(t, 1, len(mc.PayloadContent)) {
		assert.Equal(t, "example.com DoT", mc.PayloadContent[0].Name)
		assert.Equal(t, "example.com DoT", mc.PayloadContent[0].PayloadDisplayName)
		assert.Equal(t, "example.com", mc.PayloadContent[0].DNSSettings.ServerName)
	}
}
