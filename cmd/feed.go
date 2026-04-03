package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/Suoyiran1/xhs-cli/internal/xiaohongshu"
	"github.com/go-rod/rod"
	"github.com/spf13/cobra"
)

var feedCmd = &cobra.Command{
	Use:   "feed",
	Short: "获取首页推荐 Feed",
	Long: `获取小红书首页推荐内容列表。

示例:
  xhs feed --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return withPage(func(page *rod.Page) error {
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			action := xiaohongshu.NewFeedsListAction(page)
			feeds, err := action.GetFeedsList(ctx)
			if err != nil {
				return fmt.Errorf("获取 Feed 失败: %w", err)
			}

			return outputResult(cmd, map[string]interface{}{
				"count": len(feeds),
				"feeds": feeds,
			})
		})
	},
}

func init() {
	rootCmd.AddCommand(feedCmd)
}
