package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/Suoyiran1/xhs-cli/internal/xiaohongshu"
	"github.com/go-rod/rod"
	"github.com/spf13/cobra"
)

var (
	likeXsecToken string
	likeUndo      bool
)

var likeCmd = &cobra.Command{
	Use:   "like <note_id>",
	Short: "点赞笔记",
	Long: `为小红书笔记点赞或取消点赞。

示例:
  xhs like 6789abcdef --xsec-token TOKEN --json
  xhs like 6789abcdef --xsec-token TOKEN --undo --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		feedID := args[0]
		if likeXsecToken == "" {
			return fmt.Errorf("必须提供 --xsec-token 参数")
		}

		return withPage(func(page *rod.Page) error {
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			action := xiaohongshu.NewLikeAction(page)
			var err error
			if likeUndo {
				err = action.Unlike(ctx, feedID, likeXsecToken)
			} else {
				err = action.Like(ctx, feedID, likeXsecToken)
			}
			if err != nil {
				return fmt.Errorf("操作失败: %w", err)
			}

			msg := "点赞成功"
			if likeUndo {
				msg = "取消点赞成功"
			}
			return outputResult(cmd, map[string]interface{}{
				"status":  "ok",
				"feed_id": feedID,
				"message": msg,
			})
		})
	},
}

func init() {
	likeCmd.Flags().StringVar(&likeXsecToken, "xsec-token", "", "访问令牌")
	likeCmd.Flags().BoolVar(&likeUndo, "undo", false, "取消点赞")
	rootCmd.AddCommand(likeCmd)
}
