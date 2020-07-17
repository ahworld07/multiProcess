package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"fmt"
	"github.com/ahworld07/gpool"
	"github.com/akamensky/argparse"
	_ "github.com/mattn/go-sqlite3"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	//"time"
)

type MySql struct {
	Db	*sql.DB
}

func (sqObj *MySql)Crt_tb() {
	// create table if not exists
	sql_job_table := `
	CREATE TABLE IF NOT EXISTS job(
		Id INTEGER NOT NULL PRIMARY KEY,
		subJob_num INTEGER UNIQUE NOT NULL,
		command	TEXT,
		status	TEXT,
		exitCode	integer,
		retry	integer, 
		starttime	datetime,
		endtime	datetime 
	);
	`
	_, err := sqObj.Db.Exec(sql_job_table)
	if err != nil {
		panic(err)
	}
}


type jobStatusType string

// These are project or module type.
const (
	J_pending    jobStatusType = "Pending"
	J_failed    jobStatusType = "Failed"
	J_running  jobStatusType = "Running"
	J_finished  jobStatusType = "Finished"
)


func CheckCount(rows *sql.Rows) (count int) {
	count = 0
	for rows.Next() {
		count ++
	}
	if err := rows.Err(); err != nil {
		panic(err)
	}
	return count
}

func Creat_tb(shell_path string, line_unit int)(dbObj *MySql) {
	shellAbsName, _ := filepath.Abs(shell_path)
	dbpath := shellAbsName + ".db"
	conn, err := sql.Open("sqlite3", dbpath)
	CheckErr(err)
	dbObj = &MySql{Db: conn}
	dbObj.Crt_tb()

	tx, _ := dbObj.Db.Begin()
	defer tx.Rollback()
	insert_job, err := tx.Prepare("INSERT INTO job(subJob_num, command, status, retry) values(?,?,?,?)")
	CheckErr(err)

	f, err := os.Open(shellAbsName)
	if err != nil {
		panic(err)
	}
	buf := bufio.NewReader(f)

	ii := 0
	var cmd_l string = ""
	N := 0
	for {
		line, err := buf.ReadString('\n')
		if err != nil || err == io.EOF {
			break
		}

		if ii == 0{
			cmd_l = line
			ii++
		}else if ii < line_unit{
			cmd_l = cmd_l + line
			ii++
		}else{
			N++
			Nrows, err := tx.Query("select Id from job where subJob_num = ?", N)
			defer Nrows.Close()
			CheckErr(err)
			if CheckCount(Nrows)==0 {
				cmd_l = strings.TrimRight(cmd_l, "\n")
				_, _ = insert_job.Exec(N, cmd_l, J_pending, 0)
			}

			ii = 1
			cmd_l = line
		}
	}

	if ii > 0{
		N++
		Nrows, err := tx.Query("select Id from job where subJob_num = ?", N)
		defer Nrows.Close()
		CheckErr(err)
		if CheckCount(Nrows)==0 {
			_, _ = insert_job.Exec(N, cmd_l, J_pending, 0)
		}
	}

	err = tx.Commit()
	CheckErr(err)
	return
}


func GetNeed2Run(dbObj *MySql)(map[int]int){
	need2run := make(map[int]int)
	tx, _ := dbObj.Db.Begin()
	defer tx.Rollback()

	rows, err := tx.Query("select subJob_num from job where Status=? or Status=?","Pending", "Failed")
	CheckErr(err)
	defer rows.Close()
	var subJob_num int
	for rows.Next() {
		err = rows.Scan(&subJob_num)
		CheckErr(err)
		need2run[subJob_num] = 0
	}
	return need2run
}

func IlterCommand(dbObj *MySql, shell_path string, line_unit int, thred int, need2run map[int]int){
	shellAbsName, _ := filepath.Abs(shell_path)
	f, err := os.Open(shellAbsName)
	if err != nil {
		panic(err)
	}
	buf := bufio.NewReader(f)

	ii := 0
	var cmd_l string = ""
	N := 0

	pool := gpool.New(thred)

	for {
		line, err := buf.ReadString('\n')
		if err != nil || err == io.EOF {
			break
		}

		if ii == 0{
			cmd_l = line
			ii++
		}else if ii < line_unit{
			cmd_l = cmd_l + line
			ii++
		}else{
			N++
			_, ok := need2run[N]
			if ok{
				cmd_l = strings.TrimRight(cmd_l, "\n")
				pool.Add(1)
				// check
				go RunCommand(N, cmd_l, pool, dbObj)
			}

			ii = 1
			cmd_l = line
		}
	}

	if ii > 0{
		N++
		_, ok := need2run[N]
		if ok{
			pool.Add(1)
			cmd_l = strings.TrimRight(cmd_l, "\n")
			RunCommand(N, cmd_l, pool, dbObj)
		}
	}
	pool.Wait()
}


func RunCommand(N int, command_line string, pool *gpool.Pool, dbObj *MySql){
	//tx, _ := dbObj.Db.Begin()
	//defer tx.Rollback()
	now := time.Now().Format("2006-01-02 15:04:05")
	//q, err := dbObj.Db.Exec(fmt.Sprintf("UPDATE job set status=%s, starttime=%v where subJob_num=%v", "Running", now, subJob_num))
	//update_job_start, _ := tx.Prepare("UPDATE job set Status = ?, starttime = ? where subJob_num = ?")
	//update_job_end, _ := tx.Prepare("UPDATE job set Status = ?, endtime = ? where subJob_num = ?")
	//update_job_start.Exec("Running", now, N)
	//err := tx.Commit()
	//_, err := dbObj.Db.Exec(fmt.Sprintf("UPDATE job set status=%s, starttime=%v where subJob_num=%v", J_running, now, N))
	_, err := dbObj.Db.Exec("UPDATE job set status=?, starttime=? where subJob_num=?", J_running, now, N)

	CheckErr(err)
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

	err = cmd.Run()
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

	os.Stdout.WriteString(fmt.Sprintf("<<<<line:%v\texitCode:%v>>>>\n%s\n", N, exitCode, stdout))
	if exitCode == 0 {
		os.Stderr.WriteString(fmt.Sprintf("<<<<line:%v\texitCode:%v>>>>\n%s\n", N, exitCode, stderr))
	}else{
		os.Stderr.WriteString(fmt.Sprintf("<<<<line:%v\texitCode:%v>>>>\n%s\n%s\n", N, exitCode, command_line, stderr))
	}
	//os.Stdout.WriteString(fmt.Sprintf("%v",exitCode))
	//os.Stderr.WriteString(stderr)
	//fmt.Println("exitCode", exitCode)
	var lock sync.Mutex //互斥锁
	lock.Lock()

	now = time.Now().Format("2006-01-02 15:04:05")
	if exitCode == 0{
		//update_job_end.Exec(J_finished, now, N)
		_, err = dbObj.Db.Exec("UPDATE job set status=?, endtime=?, exitCode=? where subJob_num=?", J_finished, now, exitCode, N)

	}else{
		_, err = dbObj.Db.Exec("UPDATE job set status=?, endtime=?, exitCode=? where subJob_num=?", J_failed, now, exitCode, N)

	}
	lock.Unlock() //解锁

	//err = tx.Commit()
	CheckErr(err)
	pool.Done()
}

func CheckExitCode(dbObj *MySql){
	tx, _ := dbObj.Db.Begin()
	defer tx.Rollback()

	rows, err := tx.Query("select exitCode from job")
	CheckErr(err)
	defer rows.Close()

	exitCode := 0
	for rows.Next() {
		err := rows.Scan(&exitCode)
		CheckErr(err)
		if exitCode != 0{
			break
		}
	}
	os.Exit(exitCode)
}

var documents string = `辅助并发程序
                    Created by Yuan Zan(seqyuan@gmail.com)
                    Version 0.0.2 (20190522)
                    输入格式同qsub_sge的输入文件格式
1) 子进程的标准错误流和标准输出流会由此程序统一输出`

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
	//opt_r := parser.Int("r", "retry", &argparse.Options{Default: 1, Help: "Max retry times"})

	err := parser.Parse(os.Args)
	if err != nil {
		fmt.Print(parser.Usage(err))
		return
	}

	dbObj := Creat_tb(*opt_i, *opt_l)

	need2run := GetNeed2Run(dbObj)

	IlterCommand(dbObj, *opt_i, *opt_l, *opt_p, need2run)

	CheckExitCode(dbObj)
}



