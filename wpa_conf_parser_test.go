package wifimanager

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

var wpaConfParserTestData = `
asdf
network={
	ssid="ssid-1"
	psk=pw-1
}
dummy text
to make
sure
regex works
network=must fail
network=}
{
network=}

network={
	ssid="ssid-2"
	psk=pw-2
}`

func test(require *require.Assertions, networks []*WPANetwork, err error) {
	require.Nil(err)
	require.Equal(2, len(networks))

	require.Equal("ssid-1", networks[0].SSID)
	require.Equal("pw-1", networks[0].PSK)

	require.Equal("ssid-2", networks[1].SSID)
	require.Equal("pw-2", networks[1].PSK)
}

func TestParseConf(t *testing.T) {
	require := require.New(t)

	networks, err := parseConf(wpaConfParserTestData)
	test(require, networks, err)
}

func TestParseWPASupplicantConf(t *testing.T) {
	require := require.New(t)

	filename := "test-conf.txt"
	err := ioutil.WriteFile(filename, []byte(wpaConfParserTestData), 0664)
	require.Nil(err)

	networks, err := ParseWPASupplicantConf(filename)
	test(require, networks, err)

	err = os.Remove(filename)
	require.Nil(err)
}
