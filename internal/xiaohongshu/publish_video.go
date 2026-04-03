package xiaohongshu

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// PublishVideoContent 发布视频内容
type PublishVideoContent struct {
	Title        string
	Content      string
	Tags         []string
	VideoPath    string
	ScheduleTime *time.Time
	Visibility   string
	Products     []string
}

func NewPublishVideoAction(page *rod.Page) (*PublishAction, error) {
	pp := page.Timeout(300 * time.Second)

	if err := pp.Navigate(urlOfPublic); err != nil {
		return nil, errors.Wrap(err, "导航到发布页面失败")
	}

	if err := pp.WaitLoad(); err != nil {
		logrus.Warnf("等待页面加载出现问题: %v，继续尝试", err)
	}
	time.Sleep(2 * time.Second)

	if err := pp.WaitDOMStable(time.Second, 0.1); err != nil {
		logrus.Warnf("等待 DOM 稳定出现问题: %v，继续尝试", err)
	}
	time.Sleep(1 * time.Second)

	if err := mustClickPublishTab(pp, "上传视频"); err != nil {
		return nil, errors.Wrap(err, "切换到上传视频失败")
	}

	time.Sleep(1 * time.Second)
	return &PublishAction{page: pp}, nil
}

func (p *PublishAction) PublishVideo(ctx context.Context, content PublishVideoContent) error {
	if content.VideoPath == "" {
		return errors.New("视频不能为空")
	}

	page := p.page.Context(ctx)

	if err := uploadVideo(page, content.VideoPath); err != nil {
		return errors.Wrap(err, "小红书上传视频失败")
	}

	if err := submitPublishVideo(page, content.Title, content.Content, content.Tags, content.ScheduleTime, content.Visibility, content.Products); err != nil {
		return errors.Wrap(err, "小红书发布失败")
	}
	return nil
}

func uploadVideo(page *rod.Page, videoPath string) error {
	pp := page.Timeout(5 * time.Minute)

	if _, err := os.Stat(videoPath); os.IsNotExist(err) {
		return errors.Wrapf(err, "视频文件不存在: %s", videoPath)
	}

	var fileInput *rod.Element
	var err error
	fileInput, err = pp.Element(".upload-input")
	if err != nil || fileInput == nil {
		fileInput, err = pp.Element("input[type='file']")
		if err != nil || fileInput == nil {
			return errors.New("未找到视频上传输入框")
		}
	}

	fileInput.MustSetFiles(videoPath)

	btn, err := waitForPublishButtonClickable(pp)
	if err != nil {
		return err
	}
	slog.Info("视频上传/处理完成", "btn", btn)
	return nil
}

func waitForPublishButtonClickable(page *rod.Page) (*rod.Element, error) {
	maxWait := 10 * time.Minute
	interval := 1 * time.Second
	start := time.Now()
	selector := ".publish-page-publish-btn button.bg-red"

	for time.Since(start) < maxWait {
		btn, err := page.Element(selector)
		if err == nil && btn != nil {
			vis, verr := btn.Visible()
			if verr == nil && vis {
				if disabled, _ := btn.Attribute("disabled"); disabled == nil {
					if cls, _ := btn.Attribute("class"); cls != nil && !strings.Contains(*cls, "disabled") {
						return btn, nil
					}
					return btn, nil
				}
			}
		}
		time.Sleep(interval)
	}
	return nil, errors.New("等待发布按钮可点击超时")
}

func submitPublishVideo(page *rod.Page, title, content string, tags []string, scheduleTime *time.Time, visibility string, products []string) error {
	titleElem, err := page.Element("div.d-input input")
	if err != nil {
		return errors.Wrap(err, "查找标题输入框失败")
	}
	if err := titleElem.Input(title); err != nil {
		return errors.Wrap(err, "输入标题失败")
	}
	time.Sleep(1 * time.Second)

	contentElem, ok := getContentElement(page)
	if !ok {
		return errors.New("没有找到内容输入框")
	}
	if err := contentElem.Input(content); err != nil {
		return errors.Wrap(err, "输入正文失败")
	}
	if err := waitAndClickTitleInput(titleElem); err != nil {
		return err
	}
	if err := inputTags(contentElem, tags); err != nil {
		return err
	}

	time.Sleep(1 * time.Second)

	if scheduleTime != nil {
		if err := setSchedulePublish(page, *scheduleTime); err != nil {
			return errors.Wrap(err, "设置定时发布失败")
		}
	}

	if err := setVisibility(page, visibility); err != nil {
		return errors.Wrap(err, "设置可见范围失败")
	}

	if err := bindProducts(page, products); err != nil {
		return errors.Wrap(err, "绑定商品失败")
	}

	btn, err := waitForPublishButtonClickable(page)
	if err != nil {
		return err
	}

	if err := btn.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return errors.Wrap(err, "点击发布按钮失败")
	}

	time.Sleep(3 * time.Second)
	return nil
}
