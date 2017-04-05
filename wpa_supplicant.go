package wifimanager

import (
	"fmt"
	"os"
	"strings"

	"github.com/google/shlex"
	"github.com/gurupras/go-easyfiles"
	"github.com/gurupras/gocommons"
	log "github.com/sirupsen/logrus"
)

func WPAPassphrase(ssid, psk string) (string, error) {
	cmdlineStr := fmt.Sprintf("/usr/bin/wpa_passphrase %v %v", ssid, psk)
	cmdline, err := shlex.Split(cmdlineStr)
	if err != nil {
		return "", fmt.Errorf("Failed to split commandline '%v': %v", cmdlineStr, err)
	}
	ret, stdout, stderr := gocommons.Execv(cmdline[0], cmdline[1:], true)
	_ = stdout
	if ret != 0 {
		return "", fmt.Errorf("Failed to run command '%v': %v", cmdlineStr, stderr)
	}
	return strings.TrimSpace(stdout), nil
}

func (wm *WifiManager) StartWPASupplicant(iface, confPath string) error {
	err := wm.ResetWifiInterface(iface)
	if err != nil {
		return fmt.Errorf("Failed to reset wifi interface: %v", err)
	}

	cmdlineStr := fmt.Sprintf("/sbin/wpa_supplicant -Dnl80211 -i%v -c%v", iface, confPath)
	wm.wpaSupplicantCmd = wrapCmd(cmdlineStr, "wpa_supplicant")
	wm.wpaSupplicantCmd.Start()
	return nil
}

func (wm *WifiManager) StopWPASupplicant(iface string) (err error) {
	if wm.wpaSupplicantCmd != nil {
		if err = wm.wpaSupplicantCmd.Process.Kill(); err != nil {
			return fmt.Errorf("Failed to interrupt wpa_supplicant: %v\n", err)
		}
		wm.wpaSupplicantCmd.Wait()
		if !wm.wpaSupplicantCmd.ProcessState.Exited() {
			log.Warnf("Failed to wait for wpa_supplicant process to terminate")
		}
		wm.wpaSupplicantCmd = nil
	}
	return
}

func (wm *WifiManager) AddNetworkConf(ssid, password string) error {
	f, err := easyfiles.Open(wm.WPAConfPath, os.O_APPEND|os.O_WRONLY, easyfiles.GZ_FALSE)
	if err != nil {
		return fmt.Errorf("Failed to open WPA conf file to append: %v\n", err)
	}
	defer f.Close()

	if _, err = f.Seek(0, os.SEEK_END); err != nil {
		return fmt.Errorf("Failed to seek to end of WPA conf file")
	}

	writer, err := f.Writer(0)
	if err != nil {
		return fmt.Errorf("Failed to get writer to WPA conf file: %v\n", err)
	}
	defer writer.Close()
	defer writer.Flush()

	data, err := WPAPassphrase(ssid, password)
	if err != nil {
		return err
	}
	if _, err = writer.Write([]byte("\n" + data + "\n")); err != nil {
		return fmt.Errorf("Failed to update WPA conf file: %v", err)
	}
	return nil
}
