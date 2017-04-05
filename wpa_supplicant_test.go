package wifimanager

import (
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fatih/set"
	"github.com/gurupras/gocommons"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func createTestConf(require *require.Assertions, wm *WifiManager) string {
	path := "/tmp/test-start-wpa_supplicant.conf"

	// Load network info from file
	network, err := ioutil.ReadFile("test/available-ssid.conf")
	require.Nil(err)

	err = ioutil.WriteFile(path, []byte(network), 0664)
	require.Nil(err)

	return path
}

func TestWPAConfAppend(t *testing.T) {
	require := require.New(t)

	testConf := "/tmp/test-wpa-parse.conf"
	// Now append more wifi networks
	f, err := os.OpenFile(testConf, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0664)
	require.Nil(err)
	defer os.Remove(f.Name())

	wm, err := NewWifiManager(testConf)
	require.Nil(err)
	require.NotNil(wm)

	ssids := []string{"network-1", "network-2", "network-3"}
	passwords := []string{"password-1", "password-2", "password-3"}
	ssidPskMap := make(map[string]string)

	for idx, ssid := range ssids {
		password := passwords[idx]
		n, err := wm.WpaPassphrase(ssid, password)
		require.Nil(err)
		wn := ParseWPANetwork(n)
		require.NotNil(wn)
		psk := wn.PSK
		ssidPskMap[ssid] = psk
		wm.AddNetworkConf(ssid, password)
	}

	networks, err := ParseWPASupplicantConf(testConf)
	require.Nil(err)

	for _, network := range networks {
		ssid := network.SSID
		psk := network.PSK
		_, ok := ssidPskMap[ssid]
		require.True(ok)
		require.Equal(ssidPskMap[ssid], psk)
	}
}

func TestStartWPASupplicant(t *testing.T) {
	require := require.New(t)

	wm, err := NewWifiManager("/etc/wpa_supplicant/wpa_supplicant.conf")
	require.Nil(err)
	require.NotNil(wm)

	testConf := createTestConf(require, wm)
	defer os.Remove(testConf)

	err = wm.StartWpaSupplicant("wlan0", testConf)
	require.Nil(err)

	connected := false
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		start := time.Now()
		for time.Now().Sub(start) < 10*time.Second {
			connected, err = wm.IsWifiConnected()
			require.Nil(err)
			if connected {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()
	wg.Wait()
	require.True(connected, "Failed to connect to wifi")

	err = wm.StopWpaSupplicant("wlan0")
	require.Nil(err)
}

func TestStopWPASupplicant(t *testing.T) {
	require := require.New(t)

	// Get output of pgrep wpa_supplicant before test
	_, stdout, stderr := gocommons.Execv1("pgrep", "wpa_supplicant", true)
	expected := stdout

	wm, err := NewWifiManager("/etc/wpa_supplicant/wpa_supplicant.conf")
	require.Nil(err)
	require.NotNil(wm)

	testConf := createTestConf(require, wm)
	defer os.Remove(testConf)

	err = wm.StartWpaSupplicant("wlan0", testConf)
	require.Nil(err)
	err = wm.StopWpaSupplicant("wlan0")
	require.Nil(err)

	time.Sleep(1 * time.Second)
	_, stdout, stderr = gocommons.Execv1("pgrep", "wpa_supplicant", true)
	require.Equal(expected, stdout)
	require.Equal(0, len(strings.TrimSpace(stderr)), stderr)
}

func TestCurrentSSID(t *testing.T) {
	require := require.New(t)

	wm, err := NewWifiManager("/etc/wpa_supplicant/wpa_supplicant.conf")
	require.Nil(err)
	require.NotNil(wm)

	testConf := createTestConf(require, wm)
	defer os.Remove(testConf)

	networks, err := ParseWPASupplicantConf(testConf)
	require.Nil(err)
	ssidSet := set.NewNonTS()
	for _, network := range networks {
		ssidSet.Add(network.SSID)
	}
	log.Infoln("Known SSIDS:", ssidSet)

	err = wm.StartWpaSupplicant("wlan0", testConf)
	require.Nil(err)

	ssid := ""
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		start := time.Now()
		for time.Now().Sub(start) < 10*time.Second {
			ssid, err = wm.CurrentSSID("wlan0")
			require.Nil(err)
			if len(ssid) > 0 {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		err = wm.StopWpaSupplicant("wlan0")
		require.Nil(err)
	}()
	wg.Wait()
	log.Infoln("Current SSID:", ssid)
	require.True(ssidSet.Has(ssid))

}
