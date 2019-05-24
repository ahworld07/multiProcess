BASEDIR=$(shell echo `dirname $(abspath $(lastword $(MAKEFILE_LIST)))`)
build:
	docker run --rm -v $(GOPATH):/go -v $(BASEDIR):/goProject golang:1.11 bash -c 'cd /goProject && CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build multiProcess.go'
	scp -r $(BASEDIR)/multiProcess yun:~/bin/multiProcess_sqlite
	rm $(BASEDIR)/multiProcess
