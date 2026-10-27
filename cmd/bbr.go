package cmd

import (
	"fmt"
	"trojan/util"

	"github.com/spf13/cobra"
)

// bbrCmd 表示 BBR 加速管理命令
var bbrCmd = &cobra.Command{
	Use:   "bbr",
	Short: "查看或开启BBR加速",
	Run: func(cmd *cobra.Command, args []string) {
		status := util.GetBBRStatus()
		fmt.Println()
		fmt.Printf("内核版本: %s\n", util.Cyan(status.KernelVersion))
		fmt.Printf("当前拥塞控制算法: %s\n", util.Cyan(status.CurrentAlgo))

		if status.Enabled {
			fmt.Println(util.Green("当前系统已成功开启 BBR 加速!"))
			return
		}

		choice := util.LoopInput("检测到未开启 BBR，是否现在一键开启? ", []string{"立即开启 BBR", "暂不开启"}, false)
		if choice == 1 {
			fmt.Println("正在配置并启用 BBR...")
			ok, err := util.EnableBBR()
			if ok {
				fmt.Println(util.Green("BBR 加速开启成功!"))
			} else {
				fmt.Println(util.Red(fmt.Sprintf("开启 BBR 失败: %v", err)))
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(bbrCmd)
}
