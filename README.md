# Gosible v2.0

**Gosible** 是一款基于 Go 语言开发的轻量级运维自动化工具。它旨在提供类似 Ansible 的批量任务执行与文件分发能力，但摒弃了复杂的依赖和冗长的 DSL，通过一个极简的二进制文件即可实现高效的万台机器管理。

## 🚀 v2.0 新特性

* **层级配置覆盖 (Hierarchical Vars):** 支持 `Global -> Group -> Host` 三级变量覆盖。你可以为特定主机定制差异化的端口、用户或密码。
* **并发/顺序控制:** 引入 `-f (forks)` 参数。支持百级并发加速，或针对核心业务进行 `-f 1` 的严格顺序部署。
* **递归目录同步:** 内置 SFTP 逻辑，支持文件夹递归下发，并自动保持本地文件的  **权限 (Mode)** 。
* **智能输出模式:** * `status` 模式：简洁的实时进度条，适合大规模并发。
  * `detail` 模式：实时查看每台主机的标准输出，适合调试。
* **安全保障:** 引入 `-t (timeout)` 任务超时控制，防止因单台机器网络僵死导致整体任务挂起。
* **目标过滤:** 支持 `-l (limit)` 参数，无需修改配置文件即可临时过滤执行目标（IP 或组名）。

---

## 🛠 安装与部署

### 1. 编译安装

```bash
git clone https://github.com/llody55/gosible
cd gosible
CGO_ENABLED=0 go build -ldflags="-s -w" -o gosible gosible.go
sudo mv gosible /usr/local/bin/
```

---

## 📖 使用指南

### 1. 配置 Inventory (config.yaml)

支持灵活的分组与变量嵌套：

```yaml
all:
  vars:
    user: "root"                   # 全局账户
    password: "default_password"   # 全局密码
    port: 22                       # 全局端口
  groups:
    web_cluster:
      hosts:
        192.168.1.10: {}
        192.168.1.11:
          vars:
            user: "root"
            password: "admin"
            port: 2222  # 局部覆盖全局端口,自定义端口号
    db_cluster:
      vars:
        user: "root"
        password: "db_password" # 组局部变量覆盖
        port: 2222
      hosts:
        192.168.1.20: {}
```

### 2. 常用命令示例

* **批量检查系统负载 (50 并发):**

  ```bash
  gosible -i config.yaml -f 50 -m exec -a "uptime"
  ```
* **递归同步配置目录 (顺序执行 + 详细输出):**

  ```bash
  gosible -i config.yaml -f 1 -m copy -src "./configs" -dst "/etc/app/conf" -o detail
  ```
* **针对特定组执行并设置 10秒超时:**

  ```bash
  gosible -i config.yaml -l web_cluster -t 10s -a "systemctl restart nginx"
  ```

---

## 📊 参数说明

| **参数** | **说明**                                       | **默认值** |
| -------------- | ---------------------------------------------------- | ---------------- |
| `-i`         | Inventory 配置文件路径                               | `config.yaml`  |
| `-f`         | 并发数 (Forks)，设置为 1 则为顺序执行,否则并发执行   | `5`            |
| `-m`         | 任务模式：`exec`(执行命令) 或 `copy`(分发文件)   | `exec`         |
| `-o`         | 输出模式：`status`(进度条) 或 `detail`(详细结果) | `status`       |
| `-t`         | 任务超时时间 (如: 10s, 5m, 1h)                       | `30s`          |
| `-l`         | 目标过滤，支持指定 IP 或组名                         | (空)             |
| `-src/dst`   | 文件分发的源路径与目标路径                           | (空)             |

## 🤝 贡献与反馈

欢迎提交 Issue 或 Pull Request！

项目地址: https://github.com/llody55/gosible
