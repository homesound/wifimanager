package wifimanager

import (
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gurupras/gocommons"
	"github.com/stretchr/testify/require"
)

func TestStartWPASupplicant(t *testing.T) {
	require := require.New(t)

	wm, err := NewWifiManager("/etc/wpa_supplicant/wpa_supplicant.conf")
	require.Nil(err)
	require.NotNil(wm)

	network, err := wm.WpaPassphrase("club210", "winteriscoming")
	require.Nil(err)

	err = ioutil.WriteFile("/tmp/test-start-wpa_supplicant.conf", []byte(network), 0664)
	require.Nil(err)
	defer os.Remove("/tmp/test-start-wpa_supplicant.conf")

	err = wm.StartWpaSupplicant("wlan0", "/tmp/test-start-wpa_supplicant.conf")
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

}

func TestStopWPASupplicant(t *testing.T) {
	require := require.New(t)

	// Get output of pgrep wpa_supplicant before test
	ret, stdout, stderr := gocommons.Execv1("pgrep", "wpa_supplicant", true)
	require.Zero(ret)
	expected := stdout

	wm, err := NewWifiManager("/etc/wpa_supplicant/wpa_supplicant.conf")
	require.Nil(err)
	require.NotNil(wm)

	network, err := wm.WpaPassphrase("club210", "winteriscoming")
	require.Nil(err)

	err = ioutil.WriteFile("/tmp/test-start-wpa_supplicant.conf", []byte(network), 0664)
	require.Nil(err)
	defer os.Remove("/tmp/test-start-wpa_supplicant.conf")

	err = wm.StartWpaSupplicant("wlan0", "/tmp/test-start-wpa_supplicant.conf")
	require.Nil(err)
	err = wm.StopWpaSupplicant("wlan0")
	require.Nil(err)

	time.Sleep(1 * time.Second)
	ret, stdout, stderr = gocommons.Execv1("pgrep", "wpa_supplicant", true)
	require.Zero(ret)
	require.Equal(expected, stdout)
	require.Equal(0, len(strings.TrimSpace(stderr)), stderr)
}

func TestCurrentSSID(t *testing.T) {
	require := require.New(t)

	wm, err := NewWifiManager("/etc/wpa_supplicant/wpa_supplicant.conf")
	require.Nil(err)
	require.NotNil(wm)

	network, err := wm.WpaPassphrase("club210", "winteriscoming")
	require.Nil(err)

	err = ioutil.WriteFile("/tmp/test-start-wpa_supplicant.conf", []byte(network), 0664)
	require.Nil(err)
	defer os.Remove("/tmp/test-start-wpa_supplicant.conf")

	err = wm.StartWpaSupplicant("wlan0", "/tmp/test-start-wpa_supplicant.conf")
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
	}()
	wg.Wait()
	require.Equal("club210", ssid)

}
