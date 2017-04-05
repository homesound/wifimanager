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

	// Make sure the interface is up
	err = wm.IfUp(iface)
	if err != nil {
		log.Fatalf("Failed to bring wifi interface '%v' up: %v", iface, err)
	}

	for {
		wm.Lock()
		log.Debugln("Scanning for known SSIDs...")
		if ssids, err = wm.ScanForKnownSSID(); err != nil {
			log.Errorf("Failed to scan for known SSIDs: %v", err)
		} else {
			log.Infof("Known SSIDS: %v", ssids)
			now := time.Now()
			if len(ssids) > 0 && wm.hostapdCmd != nil {
				// We found a known SSID and we're in hotspot mode.
				// Get out of hotspot and start wpa_supplicant
				log.Infoln("Found known SSIDs when hotspot is running. Disable hotspot and try to connect to SSID")
				if err = wm.StopHotspot(iface); err != nil {
					log.Errorf("Failed to stop hotspot: %v", err)
				} else {
					log.Debugln("Hotspot stopped")
					if err = wm.StartWPASupplicant(iface, wm.WPAConfPath); err != nil {
						log.Errorf("Failed to start WPA supplicant: %v", err)
					} else {
						log.Debugln("WPA supplicant started")
						noKnownSSIDTimestamp = time.Now()
					}
				}
			}
			if len(ssids) == 0 && now.Sub(noKnownSSIDTimestamp) > 10*time.Second && wm.hostapdCmd == nil {
				log.Infoln("Scanning timed out. Starting hotspot")
				if err = wm.StopWPASupplicant(iface); err != nil {
					log.Errorf("Failed to stop WPA supplicant: %v", err)
				} else {
					if err = wm.StartHotspot(iface); err != nil {
						log.Errorf("Failed to start hotspot: %v", err)
					}
				}
			}
		}
		wm.Unlock()
		time.Sleep(3 * time.Second)
	}
}
