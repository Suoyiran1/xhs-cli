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
	commentXsecToken string
	replyCommentID   string
	replyUserID      string
)

var commentCmd = &cobra.Command{
	Use:   "comment <note_id> <content>",
	Short: "发表评论",
	Long: `对小红书笔记发表评论。

示例:
  xhs comment 6789abcdef "写得真好！" --xsec-token TOKEN --json`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		feedID := args[0]
		content := args[1]
		if commentXsecToken == "" {
			return fmt.Errorf("必须提供 --xsec-token 参数")
		}

		return withPage(func(page *rod.Page) error {
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			action := xiaohongshu.NewCommentFeedAction(page)
			if err := action.PostComment(ctx, feedID, commentXsecToken, content); err != nil {
				return fmt.Errorf("评论失败: %w", err)
			}

			return outputResult(cmd, map[string]interface{}{
				"status":  "ok",
				"feed_id": feedID,
				"message": "评论成功",
			})
		})
	},
}

var replyCmd = &cobra.Command{
	Use:   "reply <note_id> <content>",
	Short: "回复评论",
	Long: `回复小红书笔记下的指定评论。

示例:
  xhs reply 6789abcdef "同意！" --xsec-token TOKEN --comment-id CID --json`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		feedID := args[0]
		content := args[1]
		if commentXsecToken == "" {
			return fmt.Errorf("必须提供 --xsec-token 参数")
		}
		if replyCommentID == "" && replyUserID == "" {
			return fmt.Errorf("必须提供 --comment-id 或 --user-id")
		}

		return withPage(func(page *rod.Page) error {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			action := xiaohongshu.NewCommentFeedAction(page)
			if err := action.ReplyToComment(ctx, feedID, commentXsecToken, replyCommentID, replyUserID, content); err != nil {
				return fmt.Errorf("回复失败: %w", err)
			}

			return outputResult(cmd, map[string]interface{}{
				"status":  "ok",
				"feed_id": feedID,
				"message": "回复成功",
			})
		})
	},
}

func init() {
	commentCmd.Flags().StringVar(&commentXsecToken, "xsec-token", "", "访问令牌")
	replyCmd.Flags().StringVar(&commentXsecToken, "xsec-token", "", "访问令牌")
	replyCmd.Flags().StringVar(&replyCommentID, "comment-id", "", "目标评论 ID")
	replyCmd.Flags().StringVar(&replyUserID, "user-id", "", "目标评论用户 ID")
	rootCmd.AddCommand(commentCmd)
	rootCmd.AddCommand(replyCmd)
}
