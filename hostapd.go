package wifimanager

import (
	"fmt"
	"io/ioutil"
	"os"

	log "github.com/sirupsen/logrus"
)

func (wm *WifiManager) StartHotspot(iface string) error {
	wm.StopWPASupplicant(iface)

	err := wm.ResetWifiInterface(iface)
	if err != nil {
		return fmt.Errorf("Failed to reset wifi interface: %v", err)
	}

	if err = runCmd(fmt.Sprintf("ifconfig %s up 10.11.12.1 netmask 255.255.255.0", iface)); err != nil {
		return fmt.Errorf("StartHotspot: Failed to bring up wifi interface")
	}

	// Now that the interface is set up, run hostapd and dnsmasq
	hostapdCmdline := "/usr/sbin/hostapd /etc/hostapd/hostapd.conf"
	wm.hostapdCmd = wrapCmd(hostapdCmdline, "hostapd")
	if wm.hostapdCmd == nil {
		return fmt.Errorf("Failed to create hostapdCmd")
	}
	wm.hostapdCmd.Start()

	dnsmasqConf := fmt.Sprintf(`
no-resolv
bind-interfaces
interface=%v
dhcp-authoritative
dhcp-range=10.11.12.10,10.11.12.20,12h
`, iface)
	tmpConf, _ := ioutil.TempFile("/tmp", "dnsmasq-")
	ioutil.WriteFile(tmpConf.Name(), []byte(dnsmasqConf), 0664)
	wm.dnsmasqConf = tmpConf.Name()

	dnsmasqCmdline := fmt.Sprintf("/usr/sbin/dnsmasq --no-resolv --bind-interfaces -i %v --dhcp-authoritative --dhcp-range=10.11.12.10,10.11.12.20,12h -d -C %v", iface, tmpConf.Name())
	wm.dnsmasqCmd = wrapCmd(dnsmasqCmdline, "dnsmasq")
	if wm.dnsmasqCmd == nil {
		return fmt.Errorf("Failed to create dnsmasqCmd")
	}
	wm.dnsmasqCmd.Start()

	log.Infoln("Started hotspot")
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

	defer os.Remove(wm.dnsmasqConf)
	wm.dnsmasqConf = ""

	log.Infoln("Stopped hotspot")
	return nil
}
