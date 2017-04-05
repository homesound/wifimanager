package wifimanager

import (
	"bufio"
	"fmt"

	"github.com/google/shlex"
	simpleexec "github.com/gurupras/go-simpleexec"
	"github.com/gurupras/gocommons"
	log "github.com/sirupsen/logrus"
)

func runCmd(cmd string) error {
	cmdline, err := shlex.Split(cmd)
	if err != nil {
		return fmt.Errorf("Failed to run command '%v': %v", cmd, err)
	}
	ret, stdout, stderr := gocommons.Execv(cmdline[0], cmdline[1:], true)
	_ = stdout
	if ret != 0 {
		return fmt.Errorf("Failed to run command '%v': %v", cmd, stderr)
	}
	return nil
}

func wrapCmd(cmd string, tag string) *simpleexec.Cmd {
	command := simpleexec.ParseCmd(cmd)
	if command == nil {
		log.Errorf("Failed to parse command '%v'", cmd)
		return nil
	}
	stdout, _ := command.StdoutPipe()
	go func() {
		scanner := bufio.NewScanner(stdout)
		scanner.Split(bufio.ScanLines)
		for scanner.Scan() {
			log.Infof("%v: %v", tag, scanner.Text())
		}
	}()
	return command
}
