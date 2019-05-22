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
		Status TEXT
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
	insert_job, err := tx.Prepare("INSERT INTO job(subJob_num, command, Status) values(?,?,?)")
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
				_, _ = insert_job.Exec(N, cmd_l, J_pending)
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
			_, _ = insert_job.Exec(N, cmd_l, J_pending)
		}
	}

	err = tx.Commit()
	CheckErr(err)
	return
}

func RunCommand(N int, command_line string, pool *gpool.Pool, update_job *sql.Stmt, tx *sql.Tx){
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
	var lock sync.Mutex //互斥锁
	lock.Lock()
	if exitCode == 0{
		update_job.Exec(N, J_finished)
	}else{
		update_job.Exec(N, J_failed)
	}
	lock.Unlock() //解锁

	err = tx.Commit()
	CheckErr(err)
	pool.Done()
}

var documents string = `辅助并发程序
                    Created by Yuan Zan(seqyuan@gmail.com)
                    Version 0.0.1 (20190314)
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
	err := parser.Parse(os.Args)
	if err != nil {
		fmt.Print(parser.Usage(err))
		return
	}

	dbObj := Creat_tb(*opt_i, *opt_l)
	tx, _ := dbObj.Db.Begin()
	defer tx.Rollback()

	pool := gpool.New(*opt_p)

	rows, err := tx.Query("select subJob_num, command from job where Status=Pending or Status=Failed")
	defer rows.Close()
	var subJob_num int
	var command string
	update_job, _ := tx.Prepare("UPDATE job set Status = ? where subJob_num = ?")

	for rows.Next() {
		err = rows.Scan(&subJob_num, &command)
		CheckErr(err)
		pool.Add(1)
		go RunCommand(subJob_num, command, pool, update_job, tx)
		update_job.Exec(subJob_num, J_running)
		//err = tx.Commit()
		//CheckErr(err)
	}

	pool.Wait()
}

