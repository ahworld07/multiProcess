# multiProcess
辅助实现多线程工具

# 程序功能
> 程序适用于有很多运行时间短，但是需要运行很多的脚本，有助于减少投递的脚本。
> 例如有1000个cat 命令需要执行，这些命令间没有依赖关系，每个cat命令运行在2min左右

1. 在一个进程里并行的执行指定的命令行
2. 并行的线程可指定
3. 如果并行执行的其中某些子进程错误退出，再次执行此程序的命令可跳过成功完成的项只执行失败的子进程
4. 所有并行执行的子进程相互独立，互不影响
5. 如果并行执行的任意一个子进程退出码非0，最终multiProcess 也是非0退出

# 使用方法

## 程序参数
```
-i  --infile  Work.sh, same as qsub_sge's input format
-l  --line    Number of lines as a unit. Default: 1
-p  --thred   Thread process at same time. Default: 1
```
### -i
`-i` 参数为一个shell脚本，例如`input.sh`这个shell脚本的内容示例如下
```
echo 1
echo 11
echo 2
sddf
echo 3
grep -h
echo 4
echo 44
echo 5
echo 6
```

### -l
依照`-i`参数的示例，一共有10行命令，比如我们想每2行作为1个单位并行的执行，那么`-l`参数设置为2

### -p
如果要对整个multiProcess程序所在进程的资源做限制，可设置`-p`参数，指定最多同时并行多少个子进程

## 命令行示例
`multiProcess -i input.sh -l 2 -p 2`
我们可以把以上命令写入到`work.sh`里，然后把`work.sh`投递到SGE计算节点

## 标准输出流
multiProcess所在进程的标准输出流为其输入文件的脚本的所有子进程的标准输出流，且按照子进程的排序输出。下面是1个示例

每一个子进程的标准输出包括

`<<<<line:第几个子进程	exitCode:子进程命令行的退出码>>>>\n子进程命令行的标准输出流`

```
<<<<line:1	exitCode:0>>>>
1
11

<<<<line:2	exitCode:127>>>>
2

<<<<line:3	exitCode:2>>>>
3

<<<<line:4	exitCode:0>>>>
4
44

<<<<line:5	exitCode:0>>>>
5
6

```
## 标准错误流
multiProcess所在进程的标准错误流为其输入文件的脚本的所有子进程的标准错误流，且按照子进程的排序输出。下面是1个示例

每一个子进程的标准错误流程包括

`<<<<line:第几个子进程	exitCode:子进程命令行的退出码>>>>\n如果退出码非0则加上相应子进行的命令行\n子进程命令行的标准错误流`

```
<<<<line:1	exitCode:0>>>>

<<<<line:2	exitCode:127>>>>
echo 2
sddf
sh: sddf: 未找到命令

<<<<line:3	exitCode:2>>>>
echo 3
grep -h
用法: grep [选项]... PATTERN [FILE]...
试用‘grep --help’来获得更多信息。

<<<<line:4	exitCode:0>>>>

<<<<line:5	exitCode:0>>>>
```

## 并行子进程中其中有些子进程出错怎么办？
例如示例所示`input.sh`中的第2个和第3个子进程出错，那么待`work.sh`退出后，修正脚本的命令行，再重新运行或者投递`work.sh`即可，在重新运行
`work.sh`时，multiProcess会自动跳过已经成功完成的子进程。

# 记录子进程运行的数据库
multiProcess会针对每一个输入脚本，在其所在目录生成`脚本名称`+`.db`的sqlite3数据库，用于记录各`子进程`的运行状态，例如`input.sh`对应的数据库名称为`input.sh.db`

`input.sh.db`这个sqlite3数据库有1个名为`job`的table，`job`主要包含以下几列
```
0|Id|INTEGER|1||1
1|subJob_num|INTEGER|1||0
2|command|TEXT|0||0
3|status|TEXT|0||0
4|exitCode|integer|0||0
5|retry|integer|0||0
6|starttime|datetime|0||0
7|endtime|datetime|0||0
```
*  subJob_num 列表示记录的是第几个子进程
*  command为对应子进程的命令行
*  status表示对应子进程的状态，状态有4种:Pending Failed Running Finished
*  exitCode为对应子进程的退出码
*  retry为如果子进程出错的情况下multiProcess程序自动重新尝试运行该出错子进程的次数（目前还未启用此功能）
*  starttime为子进程开始运行的时间
*  endtime为子进程结束运行的时间

# 要在alpine docker中运行怎么办
> alpine 镜像默认不带sqlite3，multiProcess依赖于sqlite3，alpine需要更新，Dockerfile可以参考下面的

```
FROM alpine:latest

MAINTAINER Yuan Zan <seqyuan@gmail.com>

RUN apk update && apk add --no-cache \
	ttf-dejavu sqlite bash && \
	mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2 && \
	if [ -e /bin/sh ];then rm /bin/sh ; fi \
	&& if [ -e /bin/bash ];then ln -s /bin/bash /bin/sh ; fi
```
# 如果要把`multiProcess`做到docker镜像的环境变量，方便调用怎么做？
可以把`multiProcess文件`和相应的Dockerfile放到同级目录下（相同上下文），Dockerfile内容如下：

```
FROM alpine:latest

MAINTAINER Yuan Zan <seqyuan@gmail.com>

COPY ./multiProcess /opt/
WORKDIR /opt

RUN apk update && apk add --no-cache \
	ttf-dejavu sqlite bash && \
	mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2 && \
	chmod +x multiProcess
	if [ -e /bin/sh ];then rm /bin/sh ; fi \
	&& if [ -e /bin/bash ];then ln -s /bin/bash /bin/sh ; fi
	
ENV PATH /opt:$PATH:/bin
```

这样就能直接在docker容器内的命令行使用multiProcess，而不必写绝对路径了



