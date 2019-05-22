package main

import (
	//"os"
	//"bufio"
	"fmt"
	"os/exec"
	"bytes"
	"strings"
)

func main(){
	var outbuf, errbuf bytes.Buffer

	cmd := exec.Command("sh","-c","crontab -l")
	cmd.Stdout = &outbuf
	cmd.Stderr = &errbuf

	_ = cmd.Run()
	stdout := outbuf.String()
	//stderr := errbuf.String()
	//fmt.Println(stdout)
	//node_tmp := strings.Split(stdout, "\n")
	
	//for _,liness := range node_tmp{
	//	fmt.Println("....",liness)
	//}
	node_tmp := strings.Split(stdout, "\n")
	fmt.Println(node_tmp)
	addstr := ""
	for _,line := range node_tmp{
		addstr = addstr + line
	}
	fmt.Println(addstr)
	/*
	fmt.Println(strings.Index("monitor", stdout))
	addstr := "5-59/10 * * * * /annor"
	if len(node_tmp) != 0{
		addstr = addstr + "\n" + stdout
	}
	fmt.Println(addstr)
	f, err := os.Create("/Users/yuanzan/gomonitor_addCrontab")
	fmt.Println(err)
    defer f.Close()
    f.WriteString(addstr)

	//cmd2 := exec.Command("sh","-c",fmt.Sprintf("echo %s >~/gomonitor_addcrotab", addstr))
	//_ = cmd2.Run()
	cmd3 := exec.Command("sh","-c","crontab /Users/yuanzan/gomonitor_addCrontab")
	_ = cmd3.Run()
	cmd4 := exec.Command("sh","-c","rm ~/gomonitor_addCrontab")
	_ = cmd4.Run()
	*/

	f1 := 60
	fmt.Println(f1 & 0xD)

	//if f1 & 0x4:
	//	f1 = f1 | 0x8
	

}


