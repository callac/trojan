package trojan

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"trojan/core"
	"trojan/util"
)

// ControlMenu Trojan控制菜单
func ControlMenu() {
	fmt.Println()
	tType := Type()
	if tType == "trojan" {
		tType = "trojan-go"
	} else {
		tType = "trojan"
	}
	menu := []string{"启动trojan", "停止trojan", "重启trojan", "查看trojan状态", "查看trojan日志", "修改trojan端口"}
	menu = append(menu, "切换为"+tType)
	switch util.LoopInput("请选择: ", menu, true) {
	case 1:
		Start()
	case 2:
		Stop()
	case 3:
		Restart()
	case 4:
		Status(true)
	case 5:
		go util.Log("trojan", 300)
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, os.Kill)
		//阻塞
		<-c
	case 6:
		ChangePort()
	case 7:
		if err := SwitchType(tType); err != nil {
			fmt.Println(err)
		}
	}
}

// Restart 重启trojan
func Restart() {
	util.OpenPort(core.GetConfig().LocalPort)
	util.SystemctlRestart("trojan")
}

// Start 启动trojan
func Start() {
	util.OpenPort(core.GetConfig().LocalPort)
	util.SystemctlStart("trojan")
}

// Stop 停止trojan
func Stop() {
	util.SystemctlStop("trojan")
}

// Status 获取trojan状态
func Status(isPrint bool) string {
	result := util.SystemctlStatus("trojan")
	if isPrint {
		fmt.Println(result)
	}
	return result
}

// UpTime Trojan运行时间
func UpTime() string {
	result := strings.TrimSpace(util.ExecCommandWithResult("ps -Ao etime,args|grep -v grep|grep /usr/local/etc/trojan/config.json"))
	resultSlice := strings.Split(result, " ")
	if len(resultSlice) > 0 {
		return resultSlice[0]
	}
	return ""
}

// ChangePort 修改trojan端口或多端口
func ChangePort() {
	mainPort, multiPorts := GetPortInfo()
	fmt.Printf("\n当前主端口: %s\n", util.Green(strconv.Itoa(mainPort)))
	if len(multiPorts) > 0 {
		var portStrs []string
		for _, p := range multiPorts {
			portStrs = append(portStrs, strconv.Itoa(p))
		}
		fmt.Printf("当前附加多端口: %s\n", util.Green(strings.Join(portStrs, ", ")))
	} else {
		fmt.Printf("当前附加多端口: %s\n", util.Yellow("未配置"))
	}

	choice := util.LoopInput("请选择操作: ", []string{"修改主端口", "设置附加多端口(多端口同时代理)", "清空附加多端口"}, true)
	switch choice {
	case 1:
		randomPort := util.RandomPort()
		newPortStr := util.Input(fmt.Sprintf("请输入新的主端口(若使用随机端口 %s 直接回车): ", util.Blue(strconv.Itoa(randomPort))), strconv.Itoa(randomPort))
		newPort, err := strconv.Atoi(newPortStr)
		if err != nil {
			fmt.Println("修改端口失败: " + err.Error())
			return
		}
		if err := SetMainPort(newPort); err == nil {
			fmt.Println(util.Green("主端口修改成功!"))
		} else {
			fmt.Println(util.Red("修改失败: " + err.Error()))
		}
	case 2:
		inputStr := util.Input("请输入附加端口(多个端口用逗号隔开，例如 8443,2083,2087): ", "")
		if inputStr == "" {
			fmt.Println("未输入任何端口")
			return
		}
		var ports []int
		for _, s := range strings.Split(inputStr, ",") {
			s = strings.TrimSpace(s)
			if p, err := strconv.Atoi(s); err == nil && p > 0 && p <= 65535 {
				ports = append(ports, p)
			}
		}
		if err := SetMultiPorts(ports); err == nil {
			fmt.Println(util.Green("多端口重定向配置成功并已生效!"))
		} else {
			fmt.Println(util.Red("配置失败: " + err.Error()))
		}
	case 3:
		if err := SetMultiPorts([]int{}); err == nil {
			fmt.Println(util.Green("附加多端口已清空并恢复单主端口监听!"))
		} else {
			fmt.Println(util.Red("清空失败: " + err.Error()))
		}
	default:
		return
	}
}

// Version Trojan版本
func Version() string {
	flag := "-v"
	if Type() == "trojan-go" {
		flag = "-version"
	}
	result := strings.TrimSpace(util.ExecCommandWithResult("/usr/bin/trojan/trojan " + flag))
	if len(result) == 0 {
		return ""
	}
	firstLine := strings.Split(result, "\n")[0]
	tempSlice := strings.Split(firstLine, " ")
	return tempSlice[len(tempSlice)-1]
}

// SwitchType 切换Trojan类型
func SwitchType(tType string) error {
	ARCH := runtime.GOARCH
	if ARCH != "amd64" && ARCH != "arm64" {
		return errors.New("not support " + ARCH + " machine")
	}
	if tType == "trojan" && ARCH != "amd64" {
		return errors.New("trojan not support " + ARCH + " machine")
	}
	if err := core.SetValue("trojanType", tType); err != nil {
		return err
	}
	InstallTrojan("")
	return nil
}

// Type Trojan类型
func Type() string {
	tType, _ := core.GetValue("trojanType")
	if tType == "" {
		if strings.Contains(Status(false), "trojan-go") {
			tType = "trojan-go"
		} else {
			tType = "trojan"
		}
		_ = core.SetValue("trojanType", tType)
	}
	return tType
}
