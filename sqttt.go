package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"syscall"
)

func main(){
	var outbuf, errbuf bytes.Buffer

	cmd := exec.Command("sh","-c","crontab -l")
	cmd.Stdout = &outbuf
	cmd.Stderr = &errbuf

	err := cmd.Run()
	var exitCode int

	stdout := outbuf.String()
	stderr := errbuf.String()

	if err != nil {
		// try to get the exit code
		if exitError, ok := err.(*exec.ExitError); ok {
			ws := exitError.Sys().(syscall.WaitStatus)
			exitCode = ws.ExitStatus()
		} else {
			// This will happen (in OSX) if `name` is not available in $PATH,
			// in this situation, exit code could not be get, and stderr will be
			// empty string very likely, so we use the default fail code, and format err
			// to string and set to stderr
			exitCode = 1
			if stderr == "" {
				stderr = err.Error()
			}
		}
	} else {
		// success, exitCode should be 0 if go is ok
		ws := cmd.ProcessState.Sys().(syscall.WaitStatus)
		exitCode = ws.ExitStatus()
	}

	fmt.Println(exitCode)

	os.Stdout.WriteString(fmt.Sprintf("%s\nsdfsdf", stdout))
	os.Stderr.WriteString(fmt.Sprintf("%s\ndsfsdf", stderr))

	d1 := []byte("hello\ngo\n")
	_ = ioutil.WriteFile("crontab", d1, 0644)


	f, err := os.Create("crontab")
	fmt.Println(err)
	defer f.Close()
	w := bufio.NewWriter(f)
	n4, err := w.WriteString("buffered\n")
	fmt.Printf("wrote %d bytes\n", n4)
	w.Flush()

}



