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
	"github.com/homesound/go-networkmanager"
	log "github.com/sirupsen/logrus"
)

type WifiManager struct {
	WPAConfPath string
	*networkmanager.NetworkManager
	KnownSSIDs       set.Interface
	wpaSupplicantCmd *simpleexec.Cmd
	hostapdCmd       *simpleexec.Cmd
	dnsmasqCmd       *simpleexec.Cmd
	dnsmasqConf      string
	sync.Mutex
}

func New(wpaConfPath string) (*WifiManager, error) {
	if !easyfiles.Exists(wpaConfPath) {
		return nil, fmt.Errorf("WPA configuration file '%v' does not exist!", wpaConfPath)
	}
	wm := &WifiManager{}
	wm.WPAConfPath = wpaConfPath
	wm.NetworkManager = &networkmanager.NetworkManager{}
	wm.KnownSSIDs = set.New()
	if err := wm.UpdateKnownSSIDs(); err != nil {
		return nil, err
	}
	return wm, nil
}

func (wm *WifiManager) CurrentSSID(iface string) (string, error) {
	cmd := simpleexec.ParseCmd(fmt.Sprintf("/sbin/iwgetid -r %v", iface))
	buf := bytes.NewBuffer(nil)
	cmd.Stdout = buf
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(buf.String()), nil
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
				log.Debugf("Scan results=%v", scanResults)
				intersection := set.Intersection(wm.KnownSSIDs, scanSet)
				if intersection.Size() > 0 {
					for _, o := range intersection.List() {
						str := o.(string)
						ret = append(ret, str)
					}
				}
				log.Debugf("Intersection=%v", intersection)
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

func (wm *WifiManager) TestConnect(iface string, network *WPANetwork) error {
	f, err := ioutil.TempFile("/tmp", "wpa_supplicant-")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())

	confStr := network.AsConf()
	if err = ioutil.WriteFile(f.Name(), []byte(confStr), 0664); err != nil {
		return fmt.Errorf("Failed to create a temporary wpa_supllicant .conf file: %v", err)
	}

	wm.Lock()
	defer wm.Unlock()

	// Disable hostapd
	if err = wm.StopHotspot(iface); err != nil {
		return fmt.Errorf("Failed to stop hotspot to test connection: %v", err)
	}

	err = wm.StartWPASupplicant(iface, f.Name())
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
				if err == nil && strings.Compare(currentSSID, network.SSID) == 0 {
					log.Infof("Found and connected to network! SSID=%v", currentSSID)
					connected = true
					break
				} else {
					log.Warnf("SSID mismatch! expected=%v got=%v", network.SSID, currentSSID)
				}
			}
			time.Sleep(1 * time.Second)
		}
	}()
	wg.Wait()

	if err = wm.StopWPASupplicant(iface); err != nil {
		return fmt.Errorf("Failed to stop WPA supplicant: %v", err)
	}

	if connected {
		return nil
	} else {
		return fmt.Errorf("Failed to connect '%v' to SSID: %v", iface, network.SSID)
	}
}

func (wm *WifiManager) IsHostapdRunning() bool {
	return wm.hostapdCmd != nil || wm.dnsmasqCmd != nil
}

func (wm *WifiManager) IsWPASupplicantRunning() bool {
	return wm.wpaSupplicantCmd != nil
}
