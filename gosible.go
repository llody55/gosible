package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v3"
)

// --- 数据模型 ---

type Inventory struct {
	All struct {
		Vars   map[string]interface{} `yaml:"vars"`
		Groups map[string]struct {
			Vars  map[string]interface{} `yaml:"vars"`
			Hosts map[string]map[string]interface{} `yaml:"hosts"`
		} `yaml:"groups"`
	} `yaml:"all"`
}

type Target struct {
	IP       string
	Port     int
	User     string
	Password string
}

var (
	successCount int32
	failCount    int32
	printMutex   sync.Mutex // 确保输出不冲突
)

// --- 配置合并逻辑 ---

func getDeepVal(hostMap map[string]interface{}, groupVars map[string]interface{}, globalVars map[string]interface{}, key string, fallback interface{}) interface{} {
	if v, ok := hostMap["vars"]; ok {
		if vm, ok := v.(map[string]interface{}); ok {
			if val, ok := vm[key]; ok { return val }
		}
	}
	if v, ok := groupVars[key]; ok { return v }
	if v, ok := globalVars[key]; ok { return v }
	return fallback
}

func flatten(inv Inventory, limit string) []Target {
	var targets []Target
	globalVars := inv.All.Vars
	for gName, group := range inv.All.Groups {
		groupVars := group.Vars
		for ip, hostData := range group.Hosts {
			if limit != "" && !strings.Contains(ip, limit) && !strings.Contains(gName, limit) {
				continue
			}
			port := 22
			pVal := getDeepVal(hostData, groupVars, globalVars, "port", 22)
			switch v := pVal.(type) {
			case int: port = v
			case int64: port = int(v)
			}
			targets = append(targets, Target{
				IP:       ip,
				Port:     port,
				User:     fmt.Sprintf("%v", getDeepVal(hostData, groupVars, globalVars, "user", "root")),
				Password: fmt.Sprintf("%v", getDeepVal(hostData, groupVars, globalVars, "password", "")),
			})
		}
	}
	return targets
}

// --- 核心执行 ---

func run(ctx context.Context, t Target, mode, src, dst, cmd, outputMode string) error {
	config := &ssh.ClientConfig{
		User:            t.User,
		Auth:            []ssh.AuthMethod{ssh.Password(t.Password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", t.IP, t.Port))
	if err != nil { return err }

	sshConn, chans, reqs, err := ssh.NewClientConn(conn, fmt.Sprintf("%s:%d", t.IP, t.Port), config)
	if err != nil { return err }
	client := ssh.NewClient(sshConn, chans, reqs)
	defer client.Close()

	if mode == "copy" {
		sc, err := sftp.NewClient(client)
		if err != nil { return err }
		defer sc.Close()
		return copyRecursive(sc, src, dst)
	}

	session, err := client.NewSession()
	if err != nil { return err }
	defer session.Close()

	out, err := session.CombinedOutput(cmd)
	if outputMode == "detail" && err == nil {
		printSafe(fmt.Sprintf("\n✅ [%s] 输出:\n%s", t.IP, string(out)))
	}
	return err
}

func copyRecursive(sftpClient *sftp.Client, srcPath, dstPath string) error {
	return filepath.Walk(srcPath, func(path string, info os.FileInfo, err error) error {
		if err != nil { return err }
		relPath, _ := filepath.Rel(srcPath, path)
		remotePath := filepath.ToSlash(filepath.Join(dstPath, relPath))
		if info.IsDir() { return sftpClient.MkdirAll(remotePath) }
		srcFile, err := os.Open(path)
		if err != nil { return err }
		defer srcFile.Close()
		dstFile, err := sftpClient.Create(remotePath)
		if err != nil { return err }
		defer dstFile.Close()
		_, err = io.Copy(dstFile, srcFile)
		return sftpClient.Chmod(remotePath, info.Mode())
	})
}

// 安全打印函数，处理进度条刷新
func printSafe(msg string) {
	printMutex.Lock()
	defer printMutex.Unlock()
	fmt.Print(msg)
}

func updateProgress(current, total int) {
	printMutex.Lock()
	defer printMutex.Unlock()
	fmt.Printf("\r进度: [%d/%d] 成功:%d 失败:%d", current, total, atomic.LoadInt32(&successCount), atomic.LoadInt32(&failCount))
}

// --- 主程序 ---

func main() {
	invFile := flag.String("i", "inventory.yaml", "配置文件")
	forks := flag.Int("f", 5, "并发数")
	mode := flag.String("m", "exec", "模式 (exec/copy)")
	src := flag.String("src", "", "源路径")
	dst := flag.String("dst", "", "目标路径")
	cmd := flag.String("a", "uptime", "命令")
	outputMode := flag.String("o", "status", "输出模式 (status/detail)")
	timeout := flag.Duration("t", 30*time.Second, "任务超时")
	limit := flag.String("l", "", "过滤")
	flag.Parse()

	content, err := os.ReadFile(*invFile)
	if err != nil { fmt.Println("无法读取配置"); return }
	var inv Inventory
	yaml.Unmarshal(content, &inv)
	targets := flatten(inv, *limit)
	total := len(targets)

	var wg sync.WaitGroup
	sem := make(chan struct{}, *forks)

	fmt.Printf("🚀 Gosible v2.0 | 模式: %s | 目标: %d\n", *mode, total)

	for _, t := range targets {
		wg.Add(1)
		sem <- struct{}{}
		go func(target Target) {
			defer wg.Done()
			defer func() { <-sem }()

			ctx, cancel := context.WithTimeout(context.Background(), *timeout)
			defer cancel()

			err := run(ctx, target, *mode, *src, *dst, *cmd, *outputMode)
			
			if err == nil {
				atomic.AddInt32(&successCount, 1)
			} else {
				atomic.AddInt32(&failCount, 1)
				printSafe(fmt.Sprintf("\n❌ [%s:%d] 失败: %v", target.IP, target.Port, err))
			}

			if *outputMode == "status" {
				updateProgress(int(atomic.LoadInt32(&successCount)+atomic.LoadInt32(&failCount)), total)
			}
		}(t)
		if *forks == 1 { wg.Wait() }
	}

	wg.Wait()
	fmt.Printf("\n\n🏁 任务汇总: 成功 %d, 失败 %d, 总数 %d\n", successCount, failCount, total)
}