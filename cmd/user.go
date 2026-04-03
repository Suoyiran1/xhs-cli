package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/Suoyiran1/xhs-cli/internal/xiaohongshu"
	"github.com/go-rod/rod"
	"github.com/spf13/cobra"
)

var userXsecToken string

var userCmd = &cobra.Command{
	Use:   "user <user_id>",
	Short: "获取用户主页信息",
	Long: `获取小红书用户主页，包括基本信息、关注/粉丝数和笔记列表。

示例:
  xhs user 5a1234567890 --xsec-token TOKEN --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		userID := args[0]
		if userXsecToken == "" {
			return fmt.Errorf("必须提供 --xsec-token 参数")
		}

		return withPage(func(page *rod.Page) error {
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			action := xiaohongshu.NewUserProfileAction(page)
			result, err := action.UserProfile(ctx, userID, userXsecToken)
			if err != nil {
				return fmt.Errorf("获取用户信息失败: %w", err)
			}

			return outputResult(cmd, result)
		})
	},
}

func init() {
	userCmd.Flags().StringVar(&userXsecToken, "xsec-token", "", "访问令牌")
	rootCmd.AddCommand(userCmd)
}
