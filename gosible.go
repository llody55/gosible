package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/pkg/sftp"
	"github.com/urfave/cli"
	"golang.org/x/crypto/ssh"
)

type HostInfo struct {
	IP       string
	Port     string
	Username string
	Password string
}

// 主方法
func main() {
	app := cli.NewApp()
	app.Name = "SSH Tool"
	app.Usage = "Tool for SSH-based host inspection and file copy"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "hosts",
			Usage: "Path to the hosts file",
		},
		cli.StringFlag{
			Name:  "run",
			Usage: "Command or script to run on hosts",
		},
		cli.StringSliceFlag{
			Name:  "copy",
			Usage: "Local and remote file paths to copy (e.g., /local/path:/remote/path)",
		},
		cli.StringFlag{
			Name:  "group",
			Usage: "Host group to execute command on",
		},
	}

	app.Action = func(c *cli.Context) error {
		hostsFile := c.String("hosts")
		if hostsFile == "" {
			return cli.NewExitError("请指定你的hosts文件", 1)
		}
		runCommand := c.String("run")
		copyFiles := c.StringSlice("copy")
		group := c.String("group")

		// 读取 hosts 文件并解析主机信息
		groups, err := readHostsFile(hostsFile)
		if err != nil {
			log.Fatal(err)
		}

		// 获取指定的主机组
		hosts, ok := groups[group]
		if !ok {
			return cli.NewExitError("指定的组不存在", 1)
		}

		// 创建一个 WaitGroup 来等待所有巡检任务完成
		var wg sync.WaitGroup

		// 设置最大并发数量
		maxConcurrency := 5

		// 创建一个通道来控制并发执行
		concurrency := make(chan struct{}, maxConcurrency)

		// 遍历组内主机列表，为每个主机启动一个 Goroutine
		for _, host := range hosts {

			concurrency <- struct{}{} // 占用一个并发槽位

			wg.Add(1)
			go func(hostInfo HostInfo, cmd string, copyFiles []string) {
				defer func() {
					<-concurrency // 释放一个并发槽位
					wg.Done()
				}()
				fmt.Printf("[%s] 正在执行任务...\n", hostInfo.IP)
				for _, copyInfo := range copyFiles {
					localPath, remotePath := splitPaths(copyInfo)
					copyFileUsingSFTP(hostInfo, localPath, remotePath)
				}
				if cmd != "" {
					checkHost(hostInfo, cmd)
				}
				fmt.Printf("[%s] 任务完成\n", hostInfo.IP)
			}(host, runCommand, copyFiles)
		}

		// 等待所有任务完成
		wg.Wait()
		return nil
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

// 读取hosts文件并解析组信息
func readHostsFile(filename string) (map[string][]HostInfo, error) {
	groups := make(map[string][]HostInfo)
	var currentGroup string

	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// 跳过空行
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			// 解析组名
			currentGroup = line[1 : len(line)-1]
			groups[currentGroup] = []HostInfo{}
		} else {
			// 解析主机信息
			parts := strings.Split(line, ":")
			if len(parts) == 4 {
				hostInfo := HostInfo{
					IP:       parts[0],
					Port:     parts[1],
					Username: parts[2],
					Password: parts[3],
				}
				groups[currentGroup] = append(groups[currentGroup], hostInfo)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return groups, nil
}

// 用于解析 copyFiles 的内容，并从中获取本地路径和远程路径。
func splitPaths(copyInfo string) (string, string) {
	parts := strings.Split(copyInfo, ":")
	if len(parts) != 2 {
		// 处理无效的格式
		return "", ""
	}
	return parts[0], parts[1]
}

// 基于sftp进行文件复制 -- copy主方法
func copyFileUsingSFTP(hostInfo HostInfo, localFilePath, remoteFilePath string) {
	fmt.Printf("Copying file %s to host: %s:%s\n", localFilePath, hostInfo.IP, remoteFilePath)

	config := &ssh.ClientConfig{
		User: hostInfo.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(hostInfo.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// 建立SSH连接
	client, err := ssh.Dial("tcp", hostInfo.IP+":"+hostInfo.Port, config)
	if err != nil {
		fmt.Printf("Failed to connect to %s: %v\n", hostInfo.IP, err)
		return
	}
	defer client.Close()

	// 创建SFTP客户端
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		fmt.Printf("Failed to create SFTP client on %s: %v\n", hostInfo.IP, err)
		return
	}
	defer sftpClient.Close()

	// 打开本地文件
	localFile, err := os.Open(localFilePath)
	if err != nil {
		fmt.Printf("Failed to open local file: %v\n", err)
		return
	}
	defer localFile.Close()

	// 创建远程文件
	remoteFile, err := sftpClient.Create(remoteFilePath)
	if err != nil {
		fmt.Printf("Failed to create remote file on %s: %v\n", hostInfo.IP, err)
		return
	}
	defer remoteFile.Close()

	// 将本地文件拷贝到远程文件
	_, err = io.Copy(remoteFile, localFile)
	if err != nil {
		fmt.Printf("Error copying file to %s: %v\n", hostInfo.IP, err)
		return
	}

	fmt.Printf("File %s copied to %s:%s\n", localFilePath, hostInfo.IP, remoteFilePath)
}

// 基于ssh执行主要命令或者脚本
func checkHost(hostInfo HostInfo, cmd string) {
	fmt.Printf("Checking host: %s\n", hostInfo.IP)
	// 在这里执行与特定主机相关的巡检任务
	// 可以使用 SSH 连接到主机并执行巡检脚本
	// 例如，使用 golang.org/x/crypto/ssh 包来建立 SSH 连接和执行命令
	config := &ssh.ClientConfig{
		User: hostInfo.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(hostInfo.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// 建立 SSH 连接
	client, err := ssh.Dial("tcp", hostInfo.IP+":"+hostInfo.Port, config)
	if err != nil {
		fmt.Printf("Failed to connect to %s: %v\n", hostInfo.IP, err)
		return
	}
	defer client.Close()

	// 执行巡检任务，例如执行远程命令
	session, err := client.NewSession()
	if err != nil {
		fmt.Printf("Failed to create session on %s: %v\n", hostInfo.IP, err)
		return
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		fmt.Printf("Error executing command on %s: %v\n", hostInfo.IP, err)
		return
	}

	fmt.Printf("Result from %s:\n%s\n", hostInfo.IP, string(output))
}
