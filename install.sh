#!/bin/bash
# Author: Jrohy
# github: https://github.com/Jrohy/trojan

#定义操作变量, 0为否, 1为是
help=0

remove=0

update=0

# GitHub 仓库配置，默认使用当前维护仓库 callac/trojan，支持通过环境变量自定义
github_repo="${GITHUB_REPO:-callac/trojan}"
download_url="https://github.com/${github_repo}/releases/download"
version_check="https://api.github.com/repos/${github_repo}/releases/latest"
service_url="https://raw.githubusercontent.com/${github_repo}/master/asset/trojan-web.service"

[[ -e /var/lib/trojan-manager ]] && update=1

#Centos 临时取消别名
[[ -f /etc/redhat-release && -z $(echo $SHELL|grep zsh) ]] && unalias -a

[[ -z $(echo $SHELL|grep zsh) ]] && shell_way="bash" || shell_way="zsh"

#######color code########
red="31m"
green="32m"
yellow="33m"
blue="36m"
fuchsia="35m"

colorEcho(){
    color=$1
    echo -e "\033[${color}${@:2}\033[0m"
}

#######get params#########
while [[ $# > 0 ]];do
    key="$1"
    case $key in
        --remove)
        remove=1
        ;;
        -h|--help)
        help=1
        ;;
        *)
                # unknown option
        ;;
    esac
    shift # past argument or value
done
#############################

help(){
    echo "bash $0 [-h|--help] [--remove]"
    echo "  -h, --help           Show help"
    echo "      --remove         remove trojan"
    return 0
}

removeTrojan() {
    colorEcho ${yellow} "正在全面清理 Trojan 及其关联服务与数据..."

    # 1. 停止并禁用所有相关 systemd 服务及进程
    systemctl stop trojan trojan-web trojan-go 2>/dev/null
    systemctl disable trojan trojan-web trojan-go 2>/dev/null
    killall -9 trojan trojan-go 2>/dev/null
    pkill -9 -f "/usr/local/bin/trojan" 2>/dev/null
    pkill -9 -f "trojan-web" 2>/dev/null

    # 2. 移除 systemd 服务单元并重新加载
    rm -f /etc/systemd/system/trojan.service 2>/dev/null
    rm -f /etc/systemd/system/trojan-web.service 2>/dev/null
    rm -f /etc/systemd/system/trojan-go.service 2>/dev/null
    rm -f /lib/systemd/system/trojan.service 2>/dev/null
    rm -f /lib/systemd/system/trojan-web.service 2>/dev/null
    systemctl daemon-reload

    # 3. 移除可执行文件
    rm -f /usr/bin/trojan /usr/local/bin/trojan 2>/dev/null
    rm -f /usr/bin/trojan-go /usr/local/bin/trojan-go 2>/dev/null

    # 4. 移除所有配置文件与管理数据目录
    rm -rf /usr/local/etc/trojan 2>/dev/null
    rm -rf /var/lib/trojan-manager 2>/dev/null

    # 5. 移除 Docker 数据库容器与持久化数据目录
    if command -v docker >/dev/null 2>&1; then
        docker rm -f trojan-mysql trojan-mariadb trojan 2>/dev/null
    fi
    rm -rf /home/mysql /home/mariadb 2>/dev/null

    # 6. 清理多端口 iptables 转发规则
    if command -v iptables >/dev/null 2>&1; then
        while iptables -t nat -D PREROUTING -j TROJAN_MULTIPORT 2>/dev/null; do :; done
        iptables -t nat -F TROJAN_MULTIPORT 2>/dev/null
        iptables -t nat -X TROJAN_MULTIPORT 2>/dev/null
    fi

    # 7. 清理 Crontab 中与 trojan 相关的定时更新任务
    if crontab -l 2>/dev/null | grep -E 'trojan|trojan-web' >/dev/null 2>&1; then
        crontab -l 2>/dev/null | grep -v -E 'trojan|trojan-web' | crontab -
    fi

    # 8. 移除 shell 环境变量与命令补全
    sed -i '/trojan/d' ~/.bashrc ~/.zshrc 2>/dev/null
    sed -i '/trojan/d' /etc/profile /etc/bashrc 2>/dev/null
    [[ -n "$shell_way" ]] && source ~/.${shell_way}rc 2>/dev/null

    colorEcho ${green} "Trojan 及所有组件已彻底清除干净 (Uninstalled successfully)!"
}

checkSys() {
    #检查是否为Root
    [ $(id -u) != "0" ] && { colorEcho ${red} "Error: You must be root to run this script"; exit 1; }

    arch=$(uname -m 2> /dev/null)
    if [[ $arch != x86_64 && $arch != aarch64 ]];then
        colorEcho $yellow "not support $arch machine".
        exit 1
    fi

    if [[ `command -v apt-get` ]];then
        package_manager='apt-get'
    elif [[ `command -v dnf` ]];then
        package_manager='dnf'
    elif [[ `command -v yum` ]];then
        package_manager='yum'
    else
        colorEcho $red "Not support OS!"
        exit 1
    fi

    # 缺失/usr/local/bin路径时自动添加
    [[ -z `echo $PATH|grep /usr/local/bin` ]] && { echo 'export PATH=$PATH:/usr/local/bin' >> /etc/bashrc; source /etc/bashrc; }
}

#安装依赖
installDependent(){
    if [[ ${package_manager} == 'dnf' || ${package_manager} == 'yum' ]];then
        ${package_manager} install socat crontabs bash-completion -y
    else
        ${package_manager} update
        ${package_manager} install socat cron bash-completion xz-utils -y
    fi
}

setupBBR() {
    local current_algo=$(sysctl -n net.ipv4.tcp_congestion_control 2>/dev/null)
    if [[ "$current_algo" == *"bbr"* ]]; then
        colorEcho $green "系统已开启 BBR 加速"
        return 0
    fi
    colorEcho $yellow "检测到未开启 BBR，正在尝试自动开启 BBR 加速..."
    modprobe tcp_bbr >/dev/null 2>&1
    if [[ -d /etc/sysctl.d ]]; then
        echo -e "net.core.default_qdisc=fq\nnet.ipv4.tcp_congestion_control=bbr" > /etc/sysctl.d/99-bbr.conf
    else
        echo -e "\nnet.core.default_qdisc=fq\nnet.ipv4.tcp_congestion_control=bbr" >> /etc/sysctl.conf
    fi
    sysctl --system >/dev/null 2>&1 || sysctl -p /etc/sysctl.conf >/dev/null 2>&1
    local new_algo=$(sysctl -n net.ipv4.tcp_congestion_control 2>/dev/null)
    if [[ "$new_algo" == *"bbr"* ]]; then
        colorEcho $green "BBR 加速自动开启成功!"
    else
        colorEcho $yellow "BBR 配置完成，若未生效可稍后重启服务器尝试"
    fi
}

setupCron() {
    if [[ `crontab -l 2>/dev/null|grep acme` ]]; then
        if [[ -z `crontab -l 2>/dev/null|grep trojan-web` || `crontab -l 2>/dev/null|grep trojan-web|grep "&"` ]]; then
            #计算北京时间早上3点时VPS的实际时间
            origin_time_zone=$(date -R|awk '{printf"%d",$6}')
            local_time_zone=${origin_time_zone%00}
            beijing_zone=8
            beijing_update_time=3
            diff_zone=$[$beijing_zone-$local_time_zone]
            local_time=$[$beijing_update_time-$diff_zone]
            if [ $local_time -lt 0 ];then
                local_time=$[24+$local_time]
            elif [ $local_time -ge 24 ];then
                local_time=$[$local_time-24]
            fi
            crontab -l 2>/dev/null|sed '/acme.sh/d' > crontab.txt
            echo "0 ${local_time}"' * * * systemctl stop trojan-web; "/root/.acme.sh"/acme.sh --cron --home "/root/.acme.sh" > /dev/null; systemctl start trojan-web' >> crontab.txt
            crontab crontab.txt
            rm -f crontab.txt
        fi
    fi
}

installTrojan(){
    local show_tip=0
    if [[ $update == 1 ]];then
        systemctl stop trojan-web >/dev/null 2>&1
        rm -f /usr/local/bin/trojan
    fi
    lastest_version=$(curl -H 'Cache-Control: no-cache' -s "$version_check" | grep 'tag_name' | cut -d\" -f4)
    if [[ -z "$lastest_version" ]]; then
        colorEcho $yellow "提示: 在仓库 $github_repo 中未检测到已发布的 Release 版本！"
        colorEcho $yellow "正在尝试回退至上游基础版本 (或请在您的 GitHub 仓库运行 GitHub Action 发布新版 Release)..."
        lastest_version=$(curl -H 'Cache-Control: no-cache' -s "https://api.github.com/repos/Jrohy/trojan/releases/latest" | grep 'tag_name' | cut -d\" -f4)
        download_url="https://github.com/Jrohy/trojan/releases/download"
    fi

    if [[ -z "$lastest_version" ]]; then
        colorEcho $red "错误: 获取版本失败，请检查网络或在 GitHub 仓库发布 Release 资产！"
        exit 1
    fi

    echo "正在下载管理程序 `colorEcho $blue $lastest_version` 版本..."
    [[ $arch == x86_64 ]] && bin="trojan-linux-amd64" || bin="trojan-linux-arm64" 
    curl -fL "$download_url/$lastest_version/$bin" -o /usr/local/bin/trojan

    # 校验下载的二进制文件是否完整有效
    local file_size=$(wc -c < /usr/local/bin/trojan 2>/dev/null || echo 0)
    if [[ $file_size -lt 500000 ]]; then
        colorEcho $red "错误: 下载管理程序文件异常 (文件大小只有 $file_size 字节，可能未找到二进制资产)！"
        rm -f /usr/local/bin/trojan
        exit 1
    fi

    chmod +x /usr/local/bin/trojan
    if [[ ! -e /etc/systemd/system/trojan-web.service ]];then
        show_tip=1
        curl -fL $service_url -o /etc/systemd/system/trojan-web.service
        systemctl daemon-reload
        systemctl enable trojan-web
    fi
    #命令补全环境变量
    [[ -z $(grep trojan ~/.${shell_way}rc) ]] && echo "source <(trojan completion ${shell_way})" >> ~/.${shell_way}rc
    source ~/.${shell_way}rc
    if [[ $update == 0 ]];then
        colorEcho $green "安装trojan管理程序成功!\n"
        echo -e "运行命令`colorEcho $blue trojan`可进行trojan管理\n"
        /usr/local/bin/trojan
    else
        if [[ `cat /usr/local/etc/trojan/config.json|grep -w "\"db\""` ]];then
            sed -i "s/\"db\"/\"database\"/g" /usr/local/etc/trojan/config.json
            systemctl restart trojan
        fi
        /usr/local/bin/trojan upgrade db
        if [[ -z `cat /usr/local/etc/trojan/config.json|grep sni` ]];then
            /usr/local/bin/trojan upgrade config
        fi
        systemctl restart trojan-web
        colorEcho $green "更新trojan管理程序成功!\n"
    fi
    setupCron
    setupBBR
    [[ $show_tip == 1 ]] && echo "浏览器访问'`colorEcho $blue https://域名`'可在线trojan多用户管理"
}

main(){
    [[ ${help} == 1 ]] && help && return
    [[ ${remove} == 1 ]] && removeTrojan && return
    [[ $update == 0 ]] && echo "正在安装trojan管理程序.." || echo "正在更新trojan管理程序.."
    checkSys
    [[ $update == 0 ]] && installDependent
    installTrojan
}

main
