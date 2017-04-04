package wifimanager

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

var wifiManagerTestData = `
network={
	ssid="ssid-1"
	psk=pw-1
}
network={
	ssid="ssid-2"
	psk=pw-2
}
network={
	ssid="phonelab"
	key_mgmt=NONE
}`

func TestConstructor(t *testing.T) {
	require := require.New(t)

	// Try with a non-existent path
	wm, err := NewWifiManager("/path/that/does/not/exist")
	require.Nil(wm)
	require.NotNil(err)

	// Now succeed
	testConf := "test-conf.txt"
	err = ioutil.WriteFile(testConf, []byte(wifiManagerTestData), 0664)
	require.Nil(err)
	defer os.Remove(testConf)

	wm, err = NewWifiManager(testConf)
	require.NotNil(wm)
	require.Nil(err)
}

func TestUpdateKnownSSIDs(t *testing.T) {
	require := require.New(t)

	testConf := "test-conf.txt"
	err := ioutil.WriteFile(testConf, []byte(wifiManagerTestData), 0664)
	require.Nil(err)
	defer os.Remove(testConf)

	wm, err := NewWifiManager(testConf)
	require.NotNil(wm)
	require.Nil(err)

	err = wm.UpdateKnownSSIDs()
	require.Nil(err)
}

func TestScanForKnownSSID(t *testing.T) {
	require := require.New(t)

	testConf := "test-conf.txt"
	err := ioutil.WriteFile(testConf, []byte(wifiManagerTestData), 0664)
	require.Nil(err)
	defer os.Remove(testConf)

	wm, err := NewWifiManager(testConf)
	require.NotNil(wm)
	require.Nil(err)

	wm.UpdateKnownSSIDs()

	ifaces, err := wm.GetWifiInterfaces()
	require.Nil(err)

	isWifiConnected, err := wm.IsWifiConnected()
	require.Nil(err)
	if isWifiConnected {
		for _, iface := range ifaces {
			ssid, err := wm.CurrentSSID(iface)
			require.Nil(err)
			wm.KnownSSIDs.Add(ssid)
		}
	}
	ssids, err := wm.ScanForKnownSSID()
	require.Nil(err)
	require.NotNil(ssids)
}

/*
func TestConnect(t *testing.T) {
	require := require.New(t)

	wm, err := NewWifiManager(testConf)
	require.NotNil(wm)
	require.Nil(err)

	ifaces, err := wm.GetWifiInterfaces()
	require.Nil(err)

	iface := ifaces[0]

	err = wm.TestConnect(iface, "club210", "winteriscoming")
	require.Nil(err)
}
*/
