package cmd

import (
	"fmt"
	"os"

	"github.com/Suoyiran1/xhs-cli/internal/configs"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	flagHeadless bool
	flagBinPath  string
	flagJSON     bool
	flagVerbose  bool
)

var rootCmd = &cobra.Command{
	Use:   "xhs",
	Short: "小红书 CLI — 让 AI Agent 直接操作小红书",
	Long: `xhs-cli 是一个命令行工具，让 AI Agent 能直接搜索、阅读、发布和互动小红书内容。

通过浏览器自动化实现，无需逆向 API。支持：
  - 搜索笔记、查看详情和评论
  - 点赞、收藏、评论
  - 查看用户主页
  - 获取首页推荐

所有命令默认输出 JSON 格式（--json），方便 AI Agent 解析。`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		configs.InitHeadless(flagHeadless)
		configs.SetBinPath(flagBinPath)
		if flagVerbose {
			logrus.SetLevel(logrus.DebugLevel)
		} else {
			logrus.SetLevel(logrus.WarnLevel)
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&flagHeadless, "headless", true, "无头模式运行浏览器")
	rootCmd.PersistentFlags().StringVar(&flagBinPath, "bin", "", "浏览器二进制文件路径 (也可设置 ROD_BROWSER_BIN 环境变量)")
	rootCmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "输出 JSON 格式")
	rootCmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "显示详细日志")
}
