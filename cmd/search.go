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
	searchSortBy      string
	searchNoteType    string
	searchPublishTime string
	searchScope       string
	searchLocation    string
	searchLimit       int
)

var searchCmd = &cobra.Command{
	Use:   "search <keyword>",
	Short: "搜索小红书笔记",
	Long: `搜索小红书内容，支持多维筛选。

示例:
  xhs search "旅行攻略" --json
  xhs search "美食" --sort 最新 --type 视频 --json
  xhs search "编程" --time 一周内 --limit 5 --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		keyword := args[0]

		return withPage(func(page *rod.Page) error {
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			search := xiaohongshu.NewSearchAction(page)

			var filters []xiaohongshu.FilterOption
			filter := xiaohongshu.FilterOption{
				SortBy:      searchSortBy,
				NoteType:    searchNoteType,
				PublishTime: searchPublishTime,
				SearchScope: searchScope,
				Location:    searchLocation,
			}
			if filter != (xiaohongshu.FilterOption{}) {
				filters = append(filters, filter)
			}

			feeds, err := search.Search(ctx, keyword, filters...)
			if err != nil {
				return fmt.Errorf("搜索失败: %w", err)
			}

			if searchLimit > 0 && len(feeds) > searchLimit {
				feeds = feeds[:searchLimit]
			}

			return outputResult(cmd, map[string]interface{}{
				"keyword": keyword,
				"count":   len(feeds),
				"feeds":   feeds,
			})
		})
	},
}

func init() {
	searchCmd.Flags().StringVar(&searchSortBy, "sort", "", "排序: 综合|最新|最多点赞|最多评论|最多收藏")
	searchCmd.Flags().StringVar(&searchNoteType, "type", "", "类型: 不限|视频|图文")
	searchCmd.Flags().StringVar(&searchPublishTime, "time", "", "时间: 不限|一天内|一周内|半年内")
	searchCmd.Flags().StringVar(&searchScope, "scope", "", "范围: 不限|已看过|未看过|已关注")
	searchCmd.Flags().StringVar(&searchLocation, "location", "", "位置: 不限|同城|附近")
	searchCmd.Flags().IntVar(&searchLimit, "limit", 0, "限制返回数量")
	rootCmd.AddCommand(searchCmd)
}
