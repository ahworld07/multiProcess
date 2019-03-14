package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/ahworld07/gpool"
	"github.com/akamensky/argparse"
	"io"
	"log"
	"sync"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

var Fail_cmd string

func RunCommand(N int, command_line string, pool *gpool.Pool){
	//command_line_l := strings.TrimSpace(command_line)
	command_line_l := strings.TrimSuffix(command_line, "\n")
	tmp := strings.Split(command_line_l, "\n")
	var multi_cmd string
	for _,v :=range tmp{
		v = strings.TrimSpace(v)
		v = strings.TrimSuffix(v, "\n")
		if multi_cmd == ""{
			multi_cmd = v
		}else{
			multi_cmd = fmt.Sprintf("%s && %s",multi_cmd, v)
		}
	}

	defaultFailedCode := 1
	var outbuf, errbuf bytes.Buffer
	/*
	for _,v :=range tmp{
		if multi_cmd == ""{
			multi_cmd = v
		}else{
			multi_cmd = multi_cmd + ";" + v
		}
	}

	*/
	//fmt.Println("aaaa",multi_cmd)
	cmd := exec.Command("sh","-c",multi_cmd)
	cmd.Stdout = &outbuf
	cmd.Stderr = &errbuf

	err := cmd.Run()
	stdout := outbuf.String()
	stderr := errbuf.String()
	var exitCode int

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
			exitCode = defaultFailedCode
			if stderr == "" {
				stderr = err.Error()
			}
		}
	} else {
		// success, exitCode should be 0 if go is ok
		ws := cmd.ProcessState.Sys().(syscall.WaitStatus)
		exitCode = ws.ExitStatus()
	}

	os.Stdout.WriteString(fmt.Sprintf(">>>>line:%v\texitCode:%v>>>>\n%s\n", N, exitCode, stdout))
	os.Stderr.WriteString(fmt.Sprintf(">>>>line:%v\texitCode:%v>>>>\n%s\n", N, exitCode, stderr))

	//os.Stdout.WriteString(fmt.Sprintf("%v",exitCode))
	//os.Stderr.WriteString(stderr)
	//fmt.Println("exitCode", exitCode)
	if exitCode != 0{
		var lock sync.Mutex //互斥锁
		lock.Lock()
		Fail_cmd = Fail_cmd + command_line
		lock.Unlock() //解锁
	}
	pool.Done()
}

var documents string = `辅助并发程序
                    Created by Yuan Zan(seqyuan@gmail.com)
                    Version 0.0.1 (20190314)
                    输入格式同qsub_sge的输入文件格式

1) 错误退出的command会统一输出到infile+.failedCMD
2) 子进程的标准错误流和标准输出流会由此程序统一输出`

func CheckErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	parser := argparse.NewParser("multiProcess", documents)
	opt_i := parser.String("i", "infile", &argparse.Options{Required: true, Help: "Work.sh, same as qsub_sge's input format"})
	opt_l := parser.Int("l", "line", &argparse.Options{Default: 1, Help: "Number of lines as a unit"})
	opt_p := parser.Int("p", "thred", &argparse.Options{Default: 1, Help: "Thread process at same time"})
	err := parser.Parse(os.Args)
	if err != nil {
		fmt.Print(parser.Usage(err))
		return
	}

	f, err := os.Open(*opt_i)
	if err != nil {
		panic(err)
	}
	buf := bufio.NewReader(f)

	ii := 0
	var cmd_l string = ""
	pool := gpool.New(*opt_p)

	N := 0

	for {
		line, err := buf.ReadString('\n')
		if err != nil || err == io.EOF {
			break
		}

		if ii == 0{
			cmd_l = line
			ii++
		}else if ii < *opt_l{
			cmd_l = cmd_l + line
			ii++
		}else{
			pool.Add(1)
			N++
			go RunCommand(N, cmd_l, pool)
			ii = 1
			cmd_l = line
		}
	}

	if ii > 0{
		N++
		pool.Add(1)
		go RunCommand(N, cmd_l, pool)
	}

	pool.Wait()
	if Fail_cmd != "" {

		outfile := *opt_i + ".failedCMD"
		fileObj, err := os.OpenFile(outfile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
		defer fileObj.Close()
		CheckErr(err)
		_, err = io.WriteString(fileObj, Fail_cmd)
		os.Exit(1)
	}
}
