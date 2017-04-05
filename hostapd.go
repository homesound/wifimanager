package wifimanager

import (
	"fmt"
	"os"
)

func (wm *WifiManager) StartHotspot(iface string) error {
	wm.StopWpaSupplicant(iface)

	err := wm.ResetWifiInterface(iface)
	if err != nil {
		return fmt.Errorf("Failed to reset wifi interface: %v", err)
	}

	if err = runCmd(fmt.Sprintf("ifconfig %s up 192.168.1.1 netmask 255.255.255.0", iface)); err != nil {
		return fmt.Errorf("StartHotspot: Failed to bring up wifi interface")
	}

	// Now that the interface is set up, run hostapd and dnsmasq
	hostapdCmdline := "/usr/sbin/hostapd /etc/hostapd/hostapd.conf"
	wm.hostapdCmd = wrapCmd(hostapdCmdline, "hostapd")
	if wm.hostapdCmd == nil {
		return fmt.Errorf("Failed to create hostapdCmd")
	}
	wm.hostapdCmd.Start()

	dnsmasqCmdline := "/usr/sbin/dnsmasq -C /etc/dnsmasq.conf"
	wm.dnsmasqCmd = wrapCmd(dnsmasqCmdline, "dnsmasq")
	if wm.dnsmasqCmd == nil {
		return fmt.Errorf("Failed to create dnsmasqCmd")
	}
	wm.dnsmasqCmd.Start()

	return nil
}

func (wm *WifiManager) StopHotspot(iface string) error {
	if wm.hostapdCmd == nil && wm.dnsmasqCmd == nil {
		return nil
	}

	wm.hostapdCmd.Process.Signal(os.Interrupt)
	wm.dnsmasqCmd.Process.Signal(os.Interrupt)

	wm.hostapdCmd.Wait()
	wm.dnsmasqCmd.Wait()

	wm.hostapdCmd = nil
	wm.dnsmasqCmd = nil

	return nil
}
