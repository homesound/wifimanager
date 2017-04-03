package wifimanager

import (
	"time"

	log "github.com/sirupsen/logrus"
)

func (wm *WifiManager) StartSmartHotspot(iface string) error {
	// Test for wifi connection. If no wifi connection is available for
	// more than 10 seconds, then turn on the hotspot and wait until a
	// connection is available.
	var err error
	var ssids []string

	noKnownSSIDTimestamp := time.Now()

	for {
		if ssids, err = wm.ScanForKnownSSID(); err != nil {
			log.Errorf("Failed to scan for known SSIDs: %v", err)
		} else {
			now := time.Now()
			if len(ssids) > 0 && wm.hostapdCmd != nil {
				// We found a known SSID and we're in hotspot mode.
				// Get out of hotspot and start wpa_supplicant
				if err = wm.StopHotspot(iface); err != nil {
					log.Errorf("Failed to stop hotspot: %v", err)
				} else {
					if err = wm.StartWpaSupplicant(iface, wm.WPAConfPath); err != nil {
						log.Errorf("Failed to start WPA supplicant: %v", err)
					} else {
						noKnownSSIDTimestamp = time.Now()
					}
				}
			}
			if len(ssids) == 0 && now.Sub(noKnownSSIDTimestamp) > 10*time.Second && wm.hostapdCmd == nil {
				if err = wm.StopWpaSupplicant(iface); err != nil {
					log.Errorf("Failed to stop WPA supplicant: %v", err)
				} else {
					if err = wm.StartHotspot(iface); err != nil {
						log.Errorf("Failed to start hotspot: %v", err)
					}
				}
			}
		}
		time.Sleep(3 * time.Second)
	}
}
