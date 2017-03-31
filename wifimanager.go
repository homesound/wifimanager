package wifimanager

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/fatih/set"
	"github.com/gurupras/go-easyfiles"
	"github.com/homesound/go-networkmanager"
)

type WifiManager struct {
	WPAConfPath string
	*network_manager.NetworkManager
	knownSSIDs set.Interface
}

func NewWifiManager(wpaConfPath string) (*WifiManager, error) {
	if !easyfiles.Exists(wpaConfPath) {
		return nil, errors.New(fmt.Sprintf("WPA configuration file '%v' does not exist!", wpaConfPath))
	}
	return &WifiManager{wpaConfPath, &network_manager.NetworkManager{}, set.New()}, nil
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
	wm.knownSSIDs = newSet
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
				scanSet := set.NewNonTS(scanResults)
				intersection := set.Intersection(wm.knownSSIDs, scanSet)
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
				// No known wifi SSIDs found.
				// Did we encounter errors?
				if errorString.Len() > 0 {
					return nil, errors.New(errorString.String())
				} else {
					// No errors and no known SSIDs
					// legit response.
					return nil, nil
				}
			}
		} else {
			// Did not find any wifi interfaces
			err = errors.New("No wifi interface found")
			return nil, err
		}
	}
}
