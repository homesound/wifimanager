package wifimanager

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fatih/set"
	simpleexec "github.com/gurupras/go-simpleexec"
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

func getWifiInterface(wm *WifiManager) (string, error) {
	ifaces, err := wm.GetWifiInterfaces()
	if err != nil {
		return "", err
	}
	if len(ifaces) == 0 {
		return "", fmt.Errorf("No wifi interface found!")
	}
	iface := ifaces[0]
	return iface, nil
}

func pgrep(str string) (string, error) {
	cmd := simpleexec.ParseCmd(fmt.Sprintf("pgrep %v", str))
	buf := bytes.NewBuffer(nil)
	cmd.Stdout = buf
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func TestWPAConfAppend(t *testing.T) {
	require := require.New(t)

	testConf := "/tmp/test-wpa-parse.conf"
	// Now append more wifi networks
	f, err := os.OpenFile(testConf, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0664)
	require.Nil(err)
	defer os.Remove(f.Name())

	wm, err := New(testConf)
	require.Nil(err)
	require.NotNil(wm)

	ssids := []string{"network-1", "network-2", "network-3"}
	passwords := []string{"password-1", "password-2", "password-3"}
	ssidPskMap := make(map[string]string)

	for idx, ssid := range ssids {
		password := passwords[idx]
		n, err := WPAPassphrase(ssid, password)
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

	wm, err := New("/etc/wpa_supplicant/wpa_supplicant.conf")
	require.Nil(err)
	require.NotNil(wm)

	iface, err := getWifiInterface(wm)
	require.Nil(err)

	testConf := createTestConf(require, wm)
	defer os.Remove(testConf)

	err = wm.StartWPASupplicant(iface, testConf)
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

	err = wm.StopWPASupplicant(iface)
	require.Nil(err)
}

func TestStopWPASupplicant(t *testing.T) {
	require := require.New(t)

	// Get output of pgrep wpa_supplicant before test
	stdout, err := pgrep("wpa_supplicant")
	require.Nil(err)
	expected := stdout

	wm, err := New("/etc/wpa_supplicant/wpa_supplicant.conf")
	require.Nil(err)
	require.NotNil(wm)

	iface, err := getWifiInterface(wm)
	require.Nil(err)

	testConf := createTestConf(require, wm)
	defer os.Remove(testConf)

	err = wm.StartWPASupplicant(iface, testConf)
	require.Nil(err)
	err = wm.StopWPASupplicant(iface)
	require.Nil(err)

	time.Sleep(1 * time.Second)
	stdout, err = pgrep("wpa_supplicant")
	require.Nil(err)
	require.Equal(expected, stdout)
}

func TestCurrentSSID(t *testing.T) {
	require := require.New(t)

	wm, err := New("/etc/wpa_supplicant/wpa_supplicant.conf")
	require.Nil(err)
	require.NotNil(wm)

	iface, err := getWifiInterface(wm)
	require.Nil(err)

	testConf := createTestConf(require, wm)
	defer os.Remove(testConf)

	networks, err := ParseWPASupplicantConf(testConf)
	require.Nil(err)
	ssidSet := set.NewNonTS()
	for _, network := range networks {
		ssidSet.Add(network.SSID)
	}
	log.Infoln("Known SSIDS:", ssidSet)

	err = wm.StartWPASupplicant(iface, testConf)
	require.Nil(err)

	ssid := ""
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		start := time.Now()
		for time.Now().Sub(start) < 10*time.Second {
			ssid, err = wm.CurrentSSID(iface)
			require.Nil(err)
			if len(ssid) > 0 {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		err = wm.StopWPASupplicant(iface)
		require.Nil(err)
	}()
	wg.Wait()
	log.Infoln("Current SSID:", ssid)
	require.True(ssidSet.Has(ssid))

}

func TestWPAPassphrase(t *testing.T) {
	t.Parallel()
	require := require.New(t)

	expected := `
network={
	ssid="test ssid with spaces"
	#psk="hello 123"
	psk=1d2d5eb60ac569d0018f4572a324029efac83d4d4a605b6c7077fd1023715f37
}`

	str, err := WPAPassphrase("test ssid with spaces", "hello 123")
	require.Nil(err)

	require.Equal(strings.TrimSpace(expected), str, "Did not match")

	// Test with spaces

}
