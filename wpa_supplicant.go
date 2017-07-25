package wifimanager

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/gurupras/go-easyfiles"
	"github.com/gurupras/go-simpleexec"
	log "github.com/sirupsen/logrus"
)

func WPAPassphrase(ssid, psk string) (string, error) {
	var wpaBlock string
	if strings.Compare(psk, "") == 0 {
		// There is no psk..open network
		// Generate a block for this by-hand
		wpaBlock = fmt.Sprintf(`
network={
	ssid="%v"
	key_mgmt=NONE
	priority=-1
}`, ssid)
	} else {
		cmdlineStr := fmt.Sprintf(`/usr/bin/wpa_passphrase "%v" "%v"`, ssid, psk)
		cmd := simpleexec.ParseCmd(cmdlineStr)
		stdout := bytes.NewBuffer(nil)
		stderr := bytes.NewBuffer(nil)
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		if err := cmd.Start(); err != nil {
			return "", fmt.Errorf("Failed to run command :%v (stderr: %v)", err, stderr.String())
		}
		if err := cmd.Wait(); err != nil {
			return "", fmt.Errorf("Failed to wait for command :%v (stderr: %v)", err, stderr.String())
		}
		wpaBlock = stdout.String()
	}
	return strings.TrimSpace(wpaBlock), nil
}

func (wm *WifiManager) StartWPASupplicant(iface, confPath string) error {
	err := wm.ResetWifiInterface(iface)
	if err != nil {
		return fmt.Errorf("Failed to reset wifi interface: %v", err)
	}

	cmdlineStr := fmt.Sprintf("/sbin/wpa_supplicant -Dnl80211 -i%v -c%v", iface, confPath)
	wm.wpaSupplicantCmd = WrapCmd(cmdlineStr, "wpa_supplicant")
	wm.wpaSupplicantCmd.Start()
	log.Infoln("Started wpa_supplicant")
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
	log.Infoln("Stopped wpa_supplicant")
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
