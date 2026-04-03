package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Suoyiran1/xhs-cli/internal/xiaohongshu"
	"github.com/go-rod/rod"
	"github.com/spf13/cobra"
)

var (
	publishTitle      string
	publishContent    string
	publishImages     []string
	publishTags       []string
	publishSchedule   string
	publishOriginal   bool
	publishVisibility string
	publishProducts   []string
	publishVideo      string
)

var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "发布小红书笔记",
	Long: `发布小红书图文或视频笔记。

图文示例:
  xhs publish --title "标题" --content "正文" --images img1.jpg,img2.jpg --json
  xhs publish --title "标题" --content "正文" --images img1.jpg --tags 美食,旅行 --json

视频示例:
  xhs publish --title "标题" --content "正文" --video video.mp4 --json

高级选项:
  xhs publish --title "标题" --content "正文" --images img.jpg --original --visibility "仅自己可见" --json
  xhs publish --title "标题" --content "正文" --images img.jpg --schedule "2026-04-10 10:30" --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if publishTitle == "" {
			return fmt.Errorf("必须提供 --title 参数")
		}

		isVideo := publishVideo != ""
		isImage := len(publishImages) > 0

		if !isVideo && !isImage {
			return fmt.Errorf("必须提供 --images 或 --video 参数")
		}
		if isVideo && isImage {
			return fmt.Errorf("--images 和 --video 不能同时使用")
		}

		var scheduleTime *time.Time
		if publishSchedule != "" {
			t, err := time.ParseInLocation("2006-01-02 15:04", publishSchedule, time.Local)
			if err != nil {
				return fmt.Errorf("定时发布时间格式错误（正确格式: 2006-01-02 15:04）: %w", err)
			}
			scheduleTime = &t
		}

		return withPageNoHeadless(func(page *rod.Page) error {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()

			if isVideo {
				return publishVideoFn(ctx, page, cmd, scheduleTime)
			}
			return publishImageFn(ctx, page, cmd, scheduleTime)
		})
	},
}

func publishImageFn(ctx context.Context, page *rod.Page, cmd *cobra.Command, scheduleTime *time.Time) error {
	action, err := xiaohongshu.NewPublishImageAction(page)
	if err != nil {
		return fmt.Errorf("打开发布页面失败: %w", err)
	}

	// Flatten comma-separated images
	var allImages []string
	for _, img := range publishImages {
		for _, i := range strings.Split(img, ",") {
			i = strings.TrimSpace(i)
			if i != "" {
				allImages = append(allImages, i)
			}
		}
	}

	content := xiaohongshu.PublishImageContent{
		Title:        publishTitle,
		Content:      publishContent,
		Tags:         publishTags,
		ImagePaths:   allImages,
		ScheduleTime: scheduleTime,
		IsOriginal:   publishOriginal,
		Visibility:   publishVisibility,
		Products:     publishProducts,
	}

	if err := action.Publish(ctx, content); err != nil {
		return fmt.Errorf("发布失败: %w", err)
	}

	return outputResult(cmd, map[string]interface{}{
		"status":  "ok",
		"message": "图文发布成功",
		"title":   publishTitle,
	})
}

func publishVideoFn(ctx context.Context, page *rod.Page, cmd *cobra.Command, scheduleTime *time.Time) error {
	action, err := xiaohongshu.NewPublishVideoAction(page)
	if err != nil {
		return fmt.Errorf("打开发布页面失败: %w", err)
	}

	content := xiaohongshu.PublishVideoContent{
		Title:        publishTitle,
		Content:      publishContent,
		Tags:         publishTags,
		VideoPath:    publishVideo,
		ScheduleTime: scheduleTime,
		Visibility:   publishVisibility,
		Products:     publishProducts,
	}

	if err := action.PublishVideo(ctx, content); err != nil {
		return fmt.Errorf("发布失败: %w", err)
	}

	return outputResult(cmd, map[string]interface{}{
		"status":  "ok",
		"message": "视频发布成功",
		"title":   publishTitle,
	})
}

func init() {
	publishCmd.Flags().StringVar(&publishTitle, "title", "", "标题（必填，最多20字）")
	publishCmd.Flags().StringVar(&publishContent, "content", "", "正文内容")
	publishCmd.Flags().StringSliceVar(&publishImages, "images", nil, "图片路径列表（逗号分隔或多次指定）")
	publishCmd.Flags().StringVar(&publishVideo, "video", "", "视频文件路径（与 --images 互斥）")
	publishCmd.Flags().StringSliceVar(&publishTags, "tags", nil, "话题标签（逗号分隔）")
	publishCmd.Flags().StringVar(&publishSchedule, "schedule", "", "定时发布时间（格式: 2006-01-02 15:04）")
	publishCmd.Flags().BoolVar(&publishOriginal, "original", false, "声明原创")
	publishCmd.Flags().StringVar(&publishVisibility, "visibility", "", "可见范围: 公开可见|仅自己可见|仅互关好友可见")
	publishCmd.Flags().StringSliceVar(&publishProducts, "products", nil, "商品关键词（逗号分隔）")
	rootCmd.AddCommand(publishCmd)
}
