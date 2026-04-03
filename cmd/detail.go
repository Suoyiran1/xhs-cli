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
	detailXsecToken      string
	detailLoadComments   bool
	detailCommentLimit   int
	detailClickReplies   bool
	detailReplyLimit     int
	detailScrollSpeed    string
)

var detailCmd = &cobra.Command{
	Use:   "detail <note_id>",
	Short: "获取笔记详情",
	Long: `获取小红书笔记详情，包括内容、图片、互动数据和评论。

示例:
  xhs detail 6789abcdef --xsec-token TOKEN --json
  xhs detail 6789abcdef --xsec-token TOKEN --comments --comment-limit 50 --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		feedID := args[0]
		if detailXsecToken == "" {
			return fmt.Errorf("必须提供 --xsec-token 参数")
		}

		return withPage(func(page *rod.Page) error {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()

			action := xiaohongshu.NewFeedDetailAction(page)

			config := xiaohongshu.DefaultCommentLoadConfig()
			config.ClickMoreReplies = detailClickReplies
			config.ScrollSpeed = detailScrollSpeed
			if detailCommentLimit > 0 {
				config.MaxCommentItems = detailCommentLimit
			}
			if detailReplyLimit > 0 {
				config.MaxRepliesThreshold = detailReplyLimit
			}

			result, err := action.GetFeedDetail(ctx, feedID, detailXsecToken, detailLoadComments, config)
			if err != nil {
				return fmt.Errorf("获取详情失败: %w", err)
			}

			return outputResult(cmd, result)
		})
	},
}

func init() {
	detailCmd.Flags().StringVar(&detailXsecToken, "xsec-token", "", "访问令牌 (从 search 结果获取)")
	detailCmd.Flags().BoolVar(&detailLoadComments, "comments", false, "加载全部评论")
	detailCmd.Flags().IntVar(&detailCommentLimit, "comment-limit", 20, "最大评论加载数")
	detailCmd.Flags().BoolVar(&detailClickReplies, "replies", false, "展开子回复")
	detailCmd.Flags().IntVar(&detailReplyLimit, "reply-limit", 10, "跳过回复数超过此值的评论")
	detailCmd.Flags().StringVar(&detailScrollSpeed, "scroll-speed", "normal", "滚动速度: slow|normal|fast")
	rootCmd.AddCommand(detailCmd)
}
