package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Suoyiran1/xhs-cli/internal/xiaohongshu"
	"github.com/go-rod/rod"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "登录小红书（扫码登录）",
	Long:  "打开浏览器窗口，显示小红书登录二维码，扫码完成登录。Cookie 会自动保存到本地。",
	RunE: func(cmd *cobra.Command, args []string) error {
		return withPageNoHeadless(func(page *rod.Page) error {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			login := xiaohongshu.NewLogin(page)

			// Check if already logged in
			loggedIn, _ := login.CheckLoginStatus(ctx)
			if loggedIn {
				return outputResult(cmd, map[string]interface{}{
					"status":  "ok",
					"message": "已登录",
				})
			}

			fmt.Fprintln(cmd.ErrOrStderr(), "请在浏览器中扫码登录小红书...")
			fmt.Fprintln(cmd.ErrOrStderr(), "等待登录中（超时 5 分钟）...")

			success := login.WaitForLogin(ctx)
			if !success {
				return fmt.Errorf("登录超时或失败")
			}

			return outputResult(cmd, map[string]interface{}{
				"status":  "ok",
				"message": "登录成功",
			})
		})
	},
}

var loginStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "检查登录状态",
	RunE: func(cmd *cobra.Command, args []string) error {
		return withPage(func(page *rod.Page) error {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			login := xiaohongshu.NewLogin(page)
			loggedIn, err := login.CheckLoginStatus(ctx)

			result := map[string]interface{}{
				"logged_in": loggedIn,
			}
			if err != nil {
				result["error"] = err.Error()
			}

			return outputResult(cmd, result)
		})
	},
}

func init() {
	loginCmd.AddCommand(loginStatusCmd)
	rootCmd.AddCommand(loginCmd)
}

func outputResult(cmd *cobra.Command, v interface{}) error {
	if flagJSON {
		data, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
	} else {
		data, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
	}
	return nil
}
