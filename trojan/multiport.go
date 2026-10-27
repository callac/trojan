package trojan

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"trojan/core"
	"trojan/util"
)

var (
	multiPortConfigFile = "/usr/local/etc/trojan/multi_ports.json"
)

// MultiPortConfig 存储附加多端口配置
type MultiPortConfig struct {
	MultiPorts []int `json:"multi_ports"`
}

/**
 * 获取当前主端口与附加多端口配置
 * @returns mainPort 主端口, multiPorts 附加多端口列表
 */
func GetPortInfo() (int, []int) {
	config := core.GetConfig()
	mainPort := config.LocalPort
	if mainPort == 0 {
		mainPort = 443
	}

	multiPorts := loadMultiPortConfig()
	return mainPort, multiPorts
}

func loadMultiPortConfig() []int {
	if !util.IsExists(multiPortConfigFile) {
		return []int{}
	}
	content, err := os.ReadFile(multiPortConfigFile)
	if err != nil {
		return []int{}
	}
	var cfg MultiPortConfig
	if err := json.Unmarshal(content, &cfg); err != nil {
		return []int{}
	}
	return cfg.MultiPorts
}

func saveMultiPortConfig(ports []int) error {
	dir := "/usr/local/etc/trojan"
	if !util.IsExists(dir) {
		_ = os.MkdirAll(dir, 0755)
	}
	cfg := MultiPortConfig{MultiPorts: ports}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(multiPortConfigFile, data, 0644)
}

/**
 * 修改主端口
 * @param newPort 新端口号
 * @returns 错误信息
 */
func SetMainPort(newPort int) error {
	if newPort < 1 || newPort > 65535 {
		return errors.New("端口号必须在 1 到 65535 之间")
	}

	config := core.GetConfig()
	oldPort := config.LocalPort
	if oldPort == newPort {
		return nil
	}

	if !core.WritePort(newPort) {
		return errors.New("写入主端口配置失败")
	}

	util.OpenPort(newPort)
	Restart()

	// 重新应用多端口转发目标
	_ = ApplyMultiPortRules()
	return nil
}

/**
 * 设置并应用附加多端口
 * @param ports 附加端口数组
 * @returns 错误信息
 */
func SetMultiPorts(ports []int) error {
	config := core.GetConfig()
	mainPort := config.LocalPort
	if mainPort == 0 {
		mainPort = 443
	}

	// 过滤非法端口与主端口
	validPorts := make([]int, 0)
	visited := make(map[int]bool)
	for _, p := range ports {
		if p >= 1 && p <= 65535 && p != mainPort && !visited[p] {
			validPorts = append(validPorts, p)
			visited[p] = true
		}
	}

	if err := saveMultiPortConfig(validPorts); err != nil {
		return err
	}

	return ApplyMultiPortRules()
}

/**
 * 应用 iptables PREROUTING REDIRECT 规则实现多端口透明转发至主端口
 * NOTE: 使用独立的 TROJAN_MULTIPORT 链，避免影响系统其它 nat 规则
 */
func ApplyMultiPortRules() error {
	mainPort, multiPorts := GetPortInfo()

	// 初始化/清空专有 iptables 链
	util.ExecCommandWithResult("iptables -t nat -N TROJAN_MULTIPORT 2>/dev/null")
	util.ExecCommandWithResult("iptables -t nat -F TROJAN_MULTIPORT")
	// 确保 PREROUTING 关联了 TROJAN_MULTIPORT 链
	checkRule := util.ExecCommandWithResult("iptables -t nat -C PREROUTING -j TROJAN_MULTIPORT 2>/dev/null")
	if checkRule != "" || strings.Contains(checkRule, "No chain") || strings.Contains(checkRule, "Bad rule") {
		_ = util.ExecCommandWithResult("iptables -t nat -I PREROUTING 1 -j TROJAN_MULTIPORT")
	}

	if len(multiPorts) == 0 {
		return nil
	}

	var portStrs []string
	for _, p := range multiPorts {
		portStrs = append(portStrs, strconv.Itoa(p))
		util.OpenPort(p)
	}
	portList := strings.Join(portStrs, ",")

	// 将多端口的 TCP 和 UDP 重定向到 trojan 主监听端口
	tcpCmd := fmt.Sprintf("iptables -t nat -A TROJAN_MULTIPORT -p tcp -m multiport --dports %s -j REDIRECT --to-ports %d", portList, mainPort)
	udpCmd := fmt.Sprintf("iptables -t nat -A TROJAN_MULTIPORT -p udp -m multiport --dports %s -j REDIRECT --to-ports %d", portList, mainPort)
	_ = util.ExecCommandWithResult(tcpCmd)
	_ = util.ExecCommandWithResult(udpCmd)

	return nil
}
