package util

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// BBRStatus BBR加速状态结构体
type BBRStatus struct {
	Enabled       bool   `json:"enabled"`
	CurrentAlgo   string `json:"current_algorithm"`
	KernelVersion string `json:"kernel_version"`
	SupportBBR    bool   `json:"support_bbr"`
}

/**
 * 获取当前系统 Linux 内核版本与 BBR 加速状态
 * @returns BBR 状态结构体
 */
func GetBBRStatus() BBRStatus {
	kernel := strings.TrimSpace(ExecCommandWithResult("uname -r"))
	currentAlgo := strings.TrimSpace(ExecCommandWithResult("sysctl -n net.ipv4.tcp_congestion_control"))

	status := BBRStatus{
		KernelVersion: kernel,
		CurrentAlgo:   currentAlgo,
		Enabled:       strings.Contains(currentAlgo, "bbr"),
		SupportBBR:    false,
	}

	// NOTE: Linux 内核 4.9+ 原生集成 BBR 算法
	reg := regexp.MustCompile(`^(\d+)\.(\d+)`)
	matches := reg.FindStringSubmatch(kernel)
	if len(matches) >= 3 {
		major, _ := strconv.Atoi(matches[1])
		minor, _ := strconv.Atoi(matches[2])
		if major > 4 || (major == 4 && minor >= 9) {
			status.SupportBBR = true
		}
	}
	return status
}

/**
 * 自动配置并开启系统 BBR 拥塞控制算法
 * @returns 开启结果及错误信息
 */
func EnableBBR() (bool, error) {
	status := GetBBRStatus()
	if status.Enabled {
		return true, nil
	}

	if !status.SupportBBR {
		return false, errors.New("当前系统内核版本低于 4.9，原生不支持 BBR，请先升级 Linux 内核")
	}

	// 载入内核模块
	_ = ExecCommand("modprobe tcp_bbr >/dev/null 2>&1")

	bbrConfig := "# BBR Congestion Control\nnet.core.default_qdisc = fq\nnet.ipv4.tcp_congestion_control = bbr\n"
	configFile := "/etc/sysctl.d/99-bbr.conf"

	// 优先写入 /etc/sysctl.d/，若目录不存在则写入 /etc/sysctl.conf
	if IsExists("/etc/sysctl.d") {
		if err := os.WriteFile(configFile, []byte(bbrConfig), 0644); err != nil {
			return false, err
		}
	} else {
		f, err := os.OpenFile("/etc/sysctl.conf", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			return false, err
		}
		defer f.Close()
		if _, err := f.WriteString("\n" + bbrConfig); err != nil {
			return false, err
		}
	}

	// 应用 sysctl 参数使其立即生效
	_ = ExecCommand("sysctl --system >/dev/null 2>&1 || sysctl -p /etc/sysctl.conf >/dev/null 2>&1")

	// 重新校验生效状态
	newStatus := GetBBRStatus()
	if newStatus.Enabled {
		return true, nil
	}

	return false, fmt.Errorf("配置已写入，但当前拥塞控制算法仍为 %s，可能需要重启系统生效", newStatus.CurrentAlgo)
}
