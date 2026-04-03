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
	favoriteXsecToken string
	favoriteUndo      bool
)

var favoriteCmd = &cobra.Command{
	Use:   "favorite <note_id>",
	Short: "收藏笔记",
	Long: `收藏或取消收藏小红书笔记。

示例:
  xhs favorite 6789abcdef --xsec-token TOKEN --json
  xhs favorite 6789abcdef --xsec-token TOKEN --undo --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		feedID := args[0]
		if favoriteXsecToken == "" {
			return fmt.Errorf("必须提供 --xsec-token 参数")
		}

		return withPage(func(page *rod.Page) error {
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			action := xiaohongshu.NewFavoriteAction(page)
			var err error
			if favoriteUndo {
				err = action.Unfavorite(ctx, feedID, favoriteXsecToken)
			} else {
				err = action.Favorite(ctx, feedID, favoriteXsecToken)
			}
			if err != nil {
				return fmt.Errorf("操作失败: %w", err)
			}

			msg := "收藏成功"
			if favoriteUndo {
				msg = "取消收藏成功"
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
	favoriteCmd.Flags().StringVar(&favoriteXsecToken, "xsec-token", "", "访问令牌")
	favoriteCmd.Flags().BoolVar(&favoriteUndo, "undo", false, "取消收藏")
	rootCmd.AddCommand(favoriteCmd)
}
