# gosible

#### 介绍

使用 Golang 语言编写的基于 SSH 协议的工具，旨在执行远程主机文件分发和命令执行功能。

#### 安装教程

### 功能概述

1. **主机信息解析** ：

* 从指定的文件中读取主机信息（IP、端口、用户名、密码），并将其解析为 `HostInfo` 结构体。

   2.**并发执行** ：

* 采用并发模型，允许并行处理多个主机上的任务。通过 `sync.WaitGroup` 控制并发数量，避免过载远程主机。
* 使用 `sync.WaitGroup` 和通道来管理并发执行的主机任务数量，以避免过度消耗资源。

   3.**文件传输** ：

* 使用 SSH 和 SFTP 客户端（`golang.org/x/crypto/ssh` 和 `github.com/pkg/sftp` 包）实现文件在本地和远程主机之间的传输。
* `copyFileUsingSFTP` 函数负责文件传输。该函数通过 SSH 连接创建 SFTP 客户端，并将本地文件复制到远程主机。

   4.**远程命令执行** ：

* 提供功能以执行指定的命令或脚本文件在远程主机上。`checkHost` 函数使用 SSH 连接执行特定命令，并返回结果。

   5.**命令行参数** ：

* 通过命令行参数指定主机文件路径、远程命令、以及要复制的文件路径。

### 结构和主要函数

* `HostInfo` 结构体：存储远程主机的连接信息（IP、端口、用户名、密码）。
* `main` 函数：程序入口，处理命令行参数，读取主机信息，然后并发执行文件传输和远程命令执行任务。
* `readHostsFile` 函数：从文件中读取主机信息并解析成 `HostInfo` 结构体。
* `splitPaths` 函数：用于解析文件传输参数，提取本地路径和远程路径。
* `copyFileUsingSFTP` 函数：通过 SFTP 客户端实现文件传输。
* `checkHost` 函数：建立 SSH 连接并执行远程命令或脚本。

#### 使用说明

1. 目录结构

   ```
   ├── go.mod
   ├── go.sum
   ├── hosts.txt
   └── gosible.go
   ```
2. 配置hosts.txt文件

   ```
   [root@llody-dev ~/go-build]#cat hosts.txt 
   192.168.1.232:22:root:admin
   192.168.1.220:22:root:admin
   ```
3. 文件下发 --copy

   ```
   [root@llody-dev ~/go-build]#gosible --hosts ./hosts.txt --copy "/root/go-build/go.mod:/opt/go.mod"
   [192.168.1.220] 正在执行任务...
   Copying file /root/go-build/go.mod to host: 192.168.1.220:/opt/go.mod
   [192.168.1.232] 正在执行任务...
   Copying file /root/go-build/go.mod to host: 192.168.1.232:/opt/go.mod
   File /root/go-build/go.mod copied to 192.168.1.220:/opt/go.mod
   [192.168.1.220] 任务完成
   File /root/go-build/go.mod copied to 192.168.1.232:/opt/go.mod
   [192.168.1.232] 任务完成
   ```
4. 命令运行 --run

   ```
   [root@llody-dev ~/go-build]#gosible --hosts ./hosts.txt --run "ls -lah /opt/ | grep go"
   [192.168.1.220] 正在执行任务...
   Checking host: 192.168.1.220
   [192.168.1.232] 正在执行任务...
   Checking host: 192.168.1.232
   Result from 192.168.1.220:
   -rw-r--r--   1 root root  401 Oct 30 13:40 go.mod

   [192.168.1.220] 任务完成
   Result from 192.168.1.232:
   -rw-r--r--   1 root root 401 10月 30 13:40 go.mod

   [192.168.1.232] 任务完成

   [root@llody-dev ~/go-build]#gosible --hosts ./hosts.txt --run "sh demo.sh"
   [192.168.1.220] 正在执行任务...
   Checking host: 192.168.1.220
   [192.168.1.232] 正在执行任务...
   Checking host: 192.168.1.232
   Result from 192.168.1.220:

                ┏┓      ┏┓
               ┏┛┻━━━━━━┛┻┓
               ┃               ☃           ┃
               ┃  ┳┛   ┗┳ ┃
               ┃     ┻    ┃
               ┗━┓      ┏━┛
                 ┃      ┗━━━━━┓
                 ┃  神兽保佑     ┣┓
                 ┃ 永无BUG！     ┏┛
                 ┗┓┓┏━┳┓┏━━━━━┛
                  ┃┫┫ ┃┫┫
                  ┗┻┛ ┗┻┛


   [192.168.1.220] 任务完成
   Result from 192.168.1.232:

                ┏┓      ┏┓
               ┏┛┻━━━━━━┛┻┓
               ┃               ☃           ┃
               ┃  ┳┛   ┗┳ ┃
               ┃     ┻    ┃
               ┗━┓      ┏━┛
                 ┃      ┗━━━━━┓
                 ┃  神兽保佑     ┣┓
                 ┃ 永无BUG！     ┏┛
                 ┗┓┓┏━┳┓┏━━━━━┛
                  ┃┫┫ ┃┫┫
                  ┗┻┛ ┗┻┛


   [192.168.1.232] 任务完成
   ```
5. 运行脚本

#### 参与贡献

1. Fork 本仓库
2. 新建 Feat_xxx 分支
3. 提交代码
4. 新建 Pull Request

#### 特技

1. 目前实现功能有文件批量下发，脚本批量执行。
