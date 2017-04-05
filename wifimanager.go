package wifimanager

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fatih/set"
	"github.com/gurupras/go-easyfiles"
	simpleexec "github.com/gurupras/go-simpleexec"
	"github.com/gurupras/gocommons"
	"github.com/homesound/go-networkmanager"
	log "github.com/sirupsen/logrus"
)

type WifiManager struct {
	WPAConfPath string
	*network_manager.NetworkManager
	KnownSSIDs       set.Interface
	wpaSupplicantCmd *simpleexec.Cmd
	hostapdCmd       *simpleexec.Cmd
	dnsmasqCmd       *simpleexec.Cmd
	sync.Mutex
}

func NewWifiManager(wpaConfPath string) (*WifiManager, error) {
	if !easyfiles.Exists(wpaConfPath) {
		return nil, fmt.Errorf("WPA configuration file '%v' does not exist!", wpaConfPath)
	}
	wm := &WifiManager{}
	wm.WPAConfPath = wpaConfPath
	wm.NetworkManager = &network_manager.NetworkManager{}
	wm.KnownSSIDs = set.New()
	return wm, nil
}

func (wm *WifiManager) CurrentSSID(iface string) (string, error) {
	ret, stdout, stderr := gocommons.Execv1("/sbin/iwgetid", fmt.Sprintf("-r %v", iface), true)
	if ret != 0 {
		return "", fmt.Errorf("Failed to run iwgetid -r: %v", stderr)
	}
	return strings.TrimSpace(stdout), nil
}

func (wm *WifiManager) ResetWifiInterface(iface string) error {
	cmds := []string{
		fmt.Sprintf("ifconfig %v down", iface),
		fmt.Sprintf("ip addr flush %v", iface),
		fmt.Sprintf("ifconfig %v up", iface),
	}
	for _, cmd := range cmds {
		if err := runCmd(cmd); err != nil {
			return fmt.Errorf("Failed to reset wifi interface: %v", err)
		}
	}
	return nil
}

func (wm *WifiManager) UpdateKnownSSIDs() error {
	wpaNetworks, err := ParseWPASupplicantConf(wm.WPAConfPath)
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

	wm.Lock()
	defer wm.Unlock()

	// Disable hostapd
	if err = wm.StopHotspot(iface); err != nil {
		return fmt.Errorf("Failed to stop hotspot to test connection: %v", err)
	}

	err = wm.StartWpaSupplicant(iface, f.Name())
	if err != nil {
		return fmt.Errorf("Failed to start wpa supplicant: %v", err)
	}
	log.Debugln("Started test WPA supplicant")

	connected := false
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		start := time.Now()
		for time.Now().Sub(start) < 10*time.Second {
			if currentSSID, err := wm.CurrentSSID(iface); err != nil {
				log.Errorf("Failed to get current SSID: %v", err)
			} else {
				if err == nil && strings.Compare(currentSSID, ssid) == 0 {
					connected = true
					break
				} else {
					log.Warnf("SSID mismatch! expected=%v got=%v", ssid, currentSSID)
				}
			}
			time.Sleep(1 * time.Second)
		}
	}()
	wg.Wait()

	if err = wm.StopWpaSupplicant(iface); err != nil {
		return fmt.Errorf("Failed to stop WPA supplicant: %v", err)
	}

	if connected {
		return nil
	} else {
		return fmt.Errorf("Failed to connect '%v' to SSID: %v", iface, ssid)
	}
}
