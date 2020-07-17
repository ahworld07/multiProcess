BASEDIR=$(shell echo `dirname $(abspath $(lastword $(MAKEFILE_LIST)))`)
build:
	docker run --rm -v $(GOPATH):/go -v $(BASEDIR):/goProject golang:latest bash -c 'cd /goProject && CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build multiProcess.go'
	scp -r $(BASEDIR)/multiProcess 203:~/bin/multiProcess_test
#scp -r $(BASEDIR)/multiProcess yun:~/bin/multiProcess_sqlite
#	cp multiProcess /Users/yuanzan/Documents/gitlab/multiprocess/
	rm $(BASEDIR)/multiProcess
