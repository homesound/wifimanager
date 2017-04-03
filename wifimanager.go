package wifimanager

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/fatih/set"
	"github.com/gurupras/go-easyfiles"
	"github.com/gurupras/gocommons"
	"github.com/homesound/go-networkmanager"
)

type WifiManager struct {
	WPAConfPath string
	*network_manager.NetworkManager
	KnownSSIDs       set.Interface
	wpaSupplicantCmd *exec.Cmd
	hostapdCmd       *exec.Cmd
	dnsmasqCmd       *exec.Cmd
}

func NewWifiManager(wpaConfPath string) (*WifiManager, error) {
	if !easyfiles.Exists(wpaConfPath) {
		return nil, fmt.Errorf("WPA configuration file '%v' does not exist!", wpaConfPath)
	}
	return &WifiManager{wpaConfPath, &network_manager.NetworkManager{}, set.New(), nil, nil, nil}, nil
}

func (wm *WifiManager) CurrentSSID(iface string) (string, error) {
	ret, stdout, stderr := gocommons.Execv1("iwgetid", fmt.Sprintf("-r %v", iface), true)
	if ret != 0 {
		return "", fmt.Errorf("Failed to run iwgetid -r: %v", stderr)
	}
	return strings.TrimSpace(stdout), nil
}

func (wm *WifiManager) UpdateKnownSSIDs() error {
	wpaNetworks, err := network_manager.ParseWPASupplicantConf(wm.WPAConfPath)
	if err != nil {
		return err
	}
	newSet := set.New()
	for _, network := range wpaNetworks {
		newSet.Add(network.SSID)
	}
	wm.KnownSSIDs = newSet
	return nil
}

func (wm *WifiManager) ScanForKnownSSID() ([]string, error) {
	ifaces, err := wm.GetWifiInterfaces()
	if err != nil {
		// Error finding interfaces
		return nil, err
	} else {
		if len(ifaces) > 0 {
			// We found wifi interfaces
			ret := make([]string, 0)
			errorString := bytes.NewBuffer(nil)
			for _, iface := range ifaces {
				scanResults, err := wm.WifiScan(iface)
				if err != nil {
					errorString.WriteString(fmt.Sprintf("%v\n", err))
					continue
				}
				scanSet := set.NewNonTS()
				for _, entry := range scanResults {
					scanSet.Add(entry.SSID)
				}
				intersection := set.Intersection(wm.KnownSSIDs, scanSet)
				if intersection.Size() > 0 {
					for _, o := range intersection.List() {
						str := o.(string)
						ret = append(ret, str)
					}
				}
			}
			// Now check the results
			if len(ret) > 0 {
				// We found some wifi SSIDs. Ignore the errors
				// and just return the results
				return ret, nil
			} else {
				// No Known wifi SSIDs found.
				// Did we encounter errors?
				if errorString.Len() > 0 {
					return nil, fmt.Errorf(errorString.String())
				} else {
					// No errors and no Known SSIDs
					// legit response.
					return nil, nil
				}
			}
		} else {
			// Did not find any wifi interfaces
			err = fmt.Errorf("No wifi interface found")
			return nil, err
		}
	}
}

func (wm *WifiManager) TestConnect(iface, ssid, password string) error {
	f, err := ioutil.TempFile("/tmp", "wpa_supplicant-")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())

	confStr, err := wm.WpaPassphrase(ssid, password)
	if err != nil {
		return err
	}
	if err = ioutil.WriteFile(f.Name(), []byte(confStr), 0664); err != nil {
		return fmt.Errorf("Failed to create a temporary wpa_supllicant .conf file: %v", err)
	}

	err = wm.StartWpaSupplicant(iface, f.Name())
	if err != nil {
		return fmt.Errorf("Failed to start wpa supplicant: %v", err)
	}

	connected := false
	mutex := &sync.Mutex{}
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		start := time.Now()
		for time.Now().Sub(start) < 10*time.Second {
			if connected, err := wm.IsWifiConnected(); err != nil {
				fmt.Errorf("Failed to check if wifi is connected: %v", err)
			} else {
				if currentSSID, err := wm.CurrentSSID(iface); err != nil {
					fmt.Errorf("Failed to get current SSID: %v", err)
				} else {
					if err != nil && connected && strings.Compare(currentSSID, ssid) == 0 {
						mutex.Lock()
						connected = true
						mutex.Unlock()
						break
					}
					time.Sleep(1 * time.Second)
				}
			}
		}
	}()
	wg.Wait()

	if err = wm.StopWpaSupplicant(iface); err != nil {
		return err
	}

	if connected {
		return nil
	} else {
		return fmt.Errorf("Failed to connect '%v' to SSID: %v", iface, ssid)
	}
}
