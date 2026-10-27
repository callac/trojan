# trojan
![](https://img.shields.io/github/v/release/Jrohy/trojan.svg) 
![](https://img.shields.io/docker/pulls/jrohy/trojan.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/Jrohy/trojan)](https://goreportcard.com/report/github.com/Jrohy/trojan)
[![Downloads](https://img.shields.io/github/downloads/Jrohy/trojan/total.svg)](https://img.shields.io/github/downloads/Jrohy/trojan/total.svg)
[![License](https://img.shields.io/badge/license-GPL%20V3-blue.svg?longCache=true)](https://www.gnu.org/licenses/gpl-3.0.en.html)


trojan多用户管理部署程序

## 功能
- 在线web页面和命令行两种方式管理trojan多用户
- 启动 / 停止 / 重启 trojan 服务端
- 支持流量统计和流量限制
- 命令行模式管理, 支持命令补全
- 集成acme.sh证书申请
- 生成客户端配置文件
- 在线实时查看trojan日志
- 在线trojan和trojan-go随时切换
- **支持 Web 端修改主代理端口，并支持配置多个对外代理端口同时可用 (iptables PREROUTING 透明重定向)**
- **内置 BBR 拥塞控制自动加速（安装自动部署 + 命令行 + Web 一键开启）**
- **全量参数化查询重构，彻底修复 SQL 注入漏洞**
- **Docker 数据库升级至 MariaDB 10.11 LTS，支持自动拉取与守护进程自启，杜绝死循环**
- 支持trojan://分享链接和二维码分享(仅限web页面)
- 支持转化为clash订阅地址并导入到 clash
- 限制用户使用期限

## 安装方式
*trojan使用请提前准备好服务器可用的域名*  

###  a. 一键脚本安装
```bash
# 安装 / 更新
source <(curl -sL https://raw.githubusercontent.com/callac/trojan/master/install.sh)

# 卸载
source <(curl -sL https://raw.githubusercontent.com/callac/trojan/master/install.sh) --remove
```
安装完后输入 `trojan` 可进入管理程序命令行界面。  
浏览器访问 `https://你的域名` 即可在线 Web 页面管理 trojan。

### b. docker运行
1. 安装 MySQL / MariaDB (推荐 MariaDB 10.11 LTS)
```bash
docker run --name trojan-mariadb --restart=always -p 3306:3306 -v /home/mariadb:/var/lib/mysql -e MYSQL_ROOT_PASSWORD=trojan -e MYSQL_ROOT_HOST=% -e MYSQL_DATABASE=trojan -d mariadb:10.11
```
端口和 root 密码以及持久化目录都可以自定义。

2. 运行 trojan 管理容器
```bash
docker run -it -d --name trojan --net=host --restart=always --privileged callac/trojan init
```
运行完后进入容器 `docker exec -it trojan bash`, 然后输入'trojan'即可进行初始化安装   

启动web服务: `systemctl start trojan-web`   

设置自启动: `systemctl enable trojan-web`

更新管理程序: `source <(curl -sL https://git.io/trojan-install)`

## 运行截图
![avatar](asset/1.png)
![avatar](asset/2.png)

## 命令行
```
Usage:
  trojan [flags]
  trojan [command]

Available Commands:
  add           添加用户
  bbr           查看或开启BBR加速
  clean         清空指定用户流量
  completion    自动命令补全(支持bash和zsh)
  del           删除用户
  help          Help about any command
  info          用户信息列表
  log           查看trojan日志
  port          修改trojan端口
  restart       重启trojan
  start         启动trojan
  status        查看trojan状态
  stop          停止trojan
  tls           证书安装
  update        更新trojan
  updateWeb     更新trojan管理程序
  version       显示版本号
  import [path] 导入sql文件
  export [path] 导出sql文件
  web           以web方式启动

Flags:
  -h, --help   help for trojan
```

## 注意
安装完trojan后强烈建议开启BBR等加速: [one_click_script](https://github.com/jinwyp/one_click_script)  

## Thanks
感谢JetBrains提供的免费GoLand  
[![avatar](asset/jetbrains.svg)](https://jb.gg/OpenSource)
