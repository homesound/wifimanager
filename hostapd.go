package wifimanager

import (
	"fmt"
	"os"

	"github.com/google/shlex"
	"github.com/gurupras/gocommons"
)

func (wm *WifiManager) StartHotspot(iface string) error {
	wm.StopWpaSupplicant(iface)

	err := wm.ResetWifiInterface(iface)
	if err != nil {
		return fmt.Errorf("Failed to reset wifi interface: %v", err)
	}
	// Now that the interface is set up, run hostapd and dnsmasq
	hostapdCmdline, err := shlex.Split("/usr/sbin/hostapd /etc/hostapd/hostapd.conf")
	if err != nil {
		return err
	}
	wm.hostapdCmd, err = gocommons.ExecvNoWait(hostapdCmdline[0], hostapdCmdline[1:], true)
	if err != nil {
		return err
	}

	dnsmasqCmdline, err := shlex.Split("/usr/sbin/dnsmasq -C /etc/dnsmasq.conf")
	if err != nil {
		return err
	}
	wm.dnsmasqCmd, err = gocommons.ExecvNoWait(dnsmasqCmdline[0], dnsmasqCmdline[1:], true)
	if err != nil {
		return err
	}
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
