package wifimanager

import (
	"bufio"
	"fmt"
	"io"
	"sync"

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
	stderr, _ := command.StderrPipe()

	mergedChan := make(chan string, 10)
	wg := sync.WaitGroup{}
	wg.Add(2)
	stdHandler := func(stdFile io.ReadCloser) {
		defer wg.Done()
		scanner := bufio.NewScanner(stdFile)
		scanner.Split(bufio.ScanLines)
		for scanner.Scan() {
			mergedChan <- scanner.Text()
		}
	}
	go stdHandler(stdout)
	go stdHandler(stderr)
	go func() {
		for line := range mergedChan {
			log.Infof("%v: %v", tag, line)
		}
	}()

	go func() {
		wg.Wait()
		close(mergedChan)
	}()

	return command
}
