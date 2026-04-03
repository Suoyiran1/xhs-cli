package xiaohongshu

import (
	"context"
	"log/slog"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// PublishImageContent 发布图文内容
type PublishImageContent struct {
	Title        string
	Content      string
	Tags         []string
	ImagePaths   []string
	ScheduleTime *time.Time
	IsOriginal   bool
	Visibility   string
	Products     []string
}

type PublishAction struct {
	page *rod.Page
}

const (
	urlOfPublic = `https://creator.xiaohongshu.com/publish/publish?source=official`
)

func NewPublishImageAction(page *rod.Page) (*PublishAction, error) {
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

	if err := mustClickPublishTab(pp, "上传图文"); err != nil {
		return nil, err
	}

	time.Sleep(1 * time.Second)

	return &PublishAction{page: pp}, nil
}

func (p *PublishAction) Publish(ctx context.Context, content PublishImageContent) error {
	if len(content.ImagePaths) == 0 {
		return errors.New("图片不能为空")
	}

	page := p.page.Context(ctx)

	if err := uploadImages(page, content.ImagePaths); err != nil {
		return errors.Wrap(err, "小红书上传图片失败")
	}

	tags := content.Tags
	if len(tags) >= 10 {
		logrus.Warnf("标签数量超过10，截取前10个标签")
		tags = tags[:10]
	}

	logrus.Infof("发布内容: title=%s, images=%v, tags=%v, schedule=%v, original=%v, visibility=%s, products=%v",
		content.Title, len(content.ImagePaths), tags, content.ScheduleTime, content.IsOriginal, content.Visibility, content.Products)

	if err := submitPublish(page, content.Title, content.Content, tags, content.ScheduleTime, content.IsOriginal, content.Visibility, content.Products); err != nil {
		return errors.Wrap(err, "小红书发布失败")
	}

	return nil
}

func removePopCover(page *rod.Page) {
	has, elem, err := page.Has("div.d-popover")
	if err != nil {
		return
	}
	if has {
		elem.MustRemove()
	}
	clickEmptyPosition(page)
}

func clickEmptyPosition(page *rod.Page) {
	x := 380 + rand.Intn(100)
	y := 20 + rand.Intn(60)
	page.Mouse.MustMoveTo(float64(x), float64(y)).MustClick(proto.InputMouseButtonLeft)
}

func mustClickPublishTab(page *rod.Page, tabname string) error {
	page.MustElement(`div.upload-content`).MustWaitVisible()

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		tab, blocked, err := getTabElement(page, tabname)
		if err != nil {
			time.Sleep(200 * time.Millisecond)
			continue
		}
		if tab == nil {
			time.Sleep(200 * time.Millisecond)
			continue
		}
		if blocked {
			removePopCover(page)
			time.Sleep(200 * time.Millisecond)
			continue
		}
		if err := tab.Click(proto.InputMouseButtonLeft, 1); err != nil {
			time.Sleep(200 * time.Millisecond)
			continue
		}
		return nil
	}

	return errors.Errorf("没有找到发布 TAB - %s", tabname)
}

func getTabElement(page *rod.Page, tabname string) (*rod.Element, bool, error) {
	elems, err := page.Elements("div.creator-tab")
	if err != nil {
		return nil, false, err
	}

	for _, elem := range elems {
		if !isElementVisible(elem) {
			continue
		}
		text, err := elem.Text()
		if err != nil {
			continue
		}
		if strings.TrimSpace(text) != tabname {
			continue
		}
		blocked, err := isElementBlocked(elem)
		if err != nil {
			return nil, false, err
		}
		return elem, blocked, nil
	}

	return nil, false, nil
}

func isElementBlocked(elem *rod.Element) (bool, error) {
	result, err := elem.Eval(`() => {
		const rect = this.getBoundingClientRect();
		if (rect.width === 0 || rect.height === 0) return true;
		const x = rect.left + rect.width / 2;
		const y = rect.top + rect.height / 2;
		const target = document.elementFromPoint(x, y);
		return !(target === this || this.contains(target));
	}`)
	if err != nil {
		return false, err
	}
	return result.Value.Bool(), nil
}

func uploadImages(page *rod.Page, imagesPaths []string) error {
	validPaths := make([]string, 0, len(imagesPaths))
	for _, path := range imagesPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			logrus.Warnf("图片文件不存在: %s", path)
			continue
		}
		validPaths = append(validPaths, path)
	}

	for i, path := range validPaths {
		selector := `input[type="file"]`
		if i == 0 {
			selector = ".upload-input"
		}

		uploadInput, err := page.Element(selector)
		if err != nil {
			return errors.Wrapf(err, "查找上传输入框失败(第%d张)", i+1)
		}
		if err := uploadInput.SetFiles([]string{path}); err != nil {
			return errors.Wrapf(err, "上传第%d张图片失败", i+1)
		}

		if err := waitForUploadComplete(page, i+1); err != nil {
			return errors.Wrapf(err, "第%d张图片上传超时", i+1)
		}
		time.Sleep(1 * time.Second)
	}

	return nil
}

func waitForUploadComplete(page *rod.Page, expectedCount int) error {
	maxWaitTime := 60 * time.Second
	checkInterval := 500 * time.Millisecond
	start := time.Now()

	for time.Since(start) < maxWaitTime {
		uploadedImages, err := page.Elements(".img-preview-area .pr")
		if err != nil {
			time.Sleep(checkInterval)
			continue
		}
		if len(uploadedImages) >= expectedCount {
			slog.Info("图片上传完成", "count", len(uploadedImages))
			return nil
		}
		time.Sleep(checkInterval)
	}

	return errors.Errorf("第%d张图片上传超时(60s)", expectedCount)
}

func submitPublish(page *rod.Page, title, content string, tags []string, scheduleTime *time.Time, isOriginal bool, visibility string, products []string) error {
	titleElem, err := page.Element("div.d-input input")
	if err != nil {
		return errors.Wrap(err, "查找标题输入框失败")
	}
	if err := titleElem.Input(title); err != nil {
		return errors.Wrap(err, "输入标题失败")
	}

	time.Sleep(500 * time.Millisecond)
	if err := checkTitleMaxLength(page); err != nil {
		return err
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

	if err := checkContentMaxLength(page); err != nil {
		return err
	}

	if scheduleTime != nil {
		if err := setSchedulePublish(page, *scheduleTime); err != nil {
			return errors.Wrap(err, "设置定时发布失败")
		}
	}

	if err := setVisibility(page, visibility); err != nil {
		return errors.Wrap(err, "设置可见范围失败")
	}

	if isOriginal {
		if err := setOriginal(page); err != nil {
			slog.Warn("设置原创声明失败，继续发布", "error", err)
		}
	}

	if err := bindProducts(page, products); err != nil {
		return errors.Wrap(err, "绑定商品失败")
	}

	submitButton, err := page.Element(".publish-page-publish-btn button.bg-red")
	if err != nil {
		return errors.Wrap(err, "查找发布按钮失败")
	}
	if err := submitButton.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return errors.Wrap(err, "点击发布按钮失败")
	}

	time.Sleep(3 * time.Second)
	return nil
}

func waitAndClickTitleInput(titleElem *rod.Element) error {
	time.Sleep(1 * time.Second)
	return titleElem.Click(proto.InputMouseButtonLeft, 1)
}

func checkTitleMaxLength(page *rod.Page) error {
	has, elem, err := page.Has(`div.title-container div.max_suffix`)
	if err != nil || !has {
		return nil
	}
	titleLength, _ := elem.Text()
	return makeMaxLengthError(titleLength)
}

func checkContentMaxLength(page *rod.Page) error {
	has, elem, err := page.Has(`div.edit-container div.length-error`)
	if err != nil || !has {
		return nil
	}
	contentLength, _ := elem.Text()
	return makeMaxLengthError(contentLength)
}

func makeMaxLengthError(elemText string) error {
	parts := strings.Split(elemText, "/")
	if len(parts) != 2 {
		return errors.Errorf("长度超过限制: %s", elemText)
	}
	return errors.Errorf("当前输入长度为%s，最大长度为%s", parts[0], parts[1])
}

func getContentElement(page *rod.Page) (*rod.Element, bool) {
	var foundElement *rod.Element
	var found bool

	page.Race().
		Element("div.ql-editor").MustHandle(func(e *rod.Element) {
		foundElement = e
		found = true
	}).
		ElementFunc(func(page *rod.Page) (*rod.Element, error) {
			return findTextboxByPlaceholder(page)
		}).MustHandle(func(e *rod.Element) {
		foundElement = e
		found = true
	}).
		MustDo()

	return foundElement, found
}

func inputTags(contentElem *rod.Element, tags []string) error {
	if len(tags) == 0 {
		return nil
	}

	time.Sleep(1 * time.Second)

	for i := 0; i < 20; i++ {
		ka, err := contentElem.KeyActions()
		if err != nil {
			return errors.Wrap(err, "创建键盘操作失败")
		}
		if err := ka.Type(input.ArrowDown).Do(); err != nil {
			return errors.Wrap(err, "按下方向键失败")
		}
		time.Sleep(10 * time.Millisecond)
	}

	ka, err := contentElem.KeyActions()
	if err != nil {
		return errors.Wrap(err, "创建键盘操作失败")
	}
	if err := ka.Press(input.Enter).Press(input.Enter).Do(); err != nil {
		return errors.Wrap(err, "按下回车键失败")
	}

	time.Sleep(1 * time.Second)

	for _, tag := range tags {
		tag = strings.TrimLeft(tag, "#")
		if err := inputTag(contentElem, tag); err != nil {
			return errors.Wrapf(err, "输入标签[%s]失败", tag)
		}
	}
	return nil
}

func inputTag(contentElem *rod.Element, tag string) error {
	if err := contentElem.Input("#"); err != nil {
		return errors.Wrap(err, "输入#失败")
	}
	time.Sleep(200 * time.Millisecond)

	for _, char := range tag {
		if err := contentElem.Input(string(char)); err != nil {
			return errors.Wrapf(err, "输入字符[%c]失败", char)
		}
		time.Sleep(50 * time.Millisecond)
	}

	time.Sleep(1 * time.Second)

	page := contentElem.Page()
	topicContainer, err := page.Element("#creator-editor-topic-container")
	if err != nil || topicContainer == nil {
		return contentElem.Input(" ")
	}

	firstItem, err := topicContainer.Element(".item")
	if err != nil || firstItem == nil {
		return contentElem.Input(" ")
	}

	if err := firstItem.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return errors.Wrap(err, "点击标签联想选项失败")
	}
	time.Sleep(500 * time.Millisecond)
	return nil
}

func findTextboxByPlaceholder(page *rod.Page) (*rod.Element, error) {
	elements := page.MustElements("p")
	for _, elem := range elements {
		placeholder, err := elem.Attribute("data-placeholder")
		if err != nil || placeholder == nil {
			continue
		}
		if strings.Contains(*placeholder, "输入正文描述") {
			return findTextboxParent(elem)
		}
	}
	return nil, errors.New("no placeholder element found")
}

func findTextboxParent(elem *rod.Element) (*rod.Element, error) {
	currentElem := elem
	for i := 0; i < 5; i++ {
		parent, err := currentElem.Parent()
		if err != nil {
			break
		}
		role, err := parent.Attribute("role")
		if err != nil || role == nil {
			currentElem = parent
			continue
		}
		if *role == "textbox" {
			return parent, nil
		}
		currentElem = parent
	}
	return nil, errors.New("no textbox parent found")
}

func isElementVisible(elem *rod.Element) bool {
	style, err := elem.Attribute("style")
	if err == nil && style != nil {
		s := *style
		if strings.Contains(s, "left: -9999px") || strings.Contains(s, "display: none") || strings.Contains(s, "visibility: hidden") {
			return false
		}
	}
	visible, err := elem.Visible()
	if err != nil {
		return true
	}
	return visible
}

func setVisibility(page *rod.Page, visibility string) error {
	if visibility == "" || visibility == "公开可见" {
		return nil
	}

	supported := map[string]bool{"仅自己可见": true, "仅互关好友可见": true}
	if !supported[visibility] {
		return errors.Errorf("不支持的可见范围: %s", visibility)
	}

	dropdown, err := page.Element("div.permission-card-wrapper div.d-select-content")
	if err != nil {
		return errors.Wrap(err, "查找可见范围下拉框失败")
	}
	if err := dropdown.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return errors.Wrap(err, "点击可见范围下拉框失败")
	}
	time.Sleep(500 * time.Millisecond)

	opts, err := page.Elements("div.d-options-wrapper div.d-grid-item div.custom-option")
	if err != nil {
		return errors.Wrap(err, "查找可见范围选项失败")
	}
	for _, opt := range opts {
		text, _ := opt.Text()
		if strings.Contains(text, visibility) {
			if err := opt.Click(proto.InputMouseButtonLeft, 1); err != nil {
				return errors.Wrap(err, "选择可见范围失败")
			}
			time.Sleep(200 * time.Millisecond)
			return nil
		}
	}
	return errors.Errorf("未找到可见范围选项: %s", visibility)
}

func setSchedulePublish(page *rod.Page, t time.Time) error {
	switchElem, err := page.Element(".post-time-wrapper .d-switch")
	if err != nil {
		return errors.Wrap(err, "查找定时发布开关失败")
	}
	if err := switchElem.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return errors.Wrap(err, "点击定时发布开关失败")
	}
	time.Sleep(800 * time.Millisecond)

	dateInput, err := page.Element(".date-picker-container input")
	if err != nil {
		return errors.Wrap(err, "查找日期时间输入框失败")
	}
	if err := dateInput.SelectAllText(); err != nil {
		return errors.Wrap(err, "选择日期时间文本失败")
	}
	if err := dateInput.Input(t.Format("2006-01-02 15:04")); err != nil {
		return errors.Wrap(err, "输入日期时间失败")
	}
	time.Sleep(500 * time.Millisecond)
	return nil
}

func setOriginal(page *rod.Page) error {
	switchCards, err := page.Elements("div.custom-switch-card")
	if err != nil {
		return errors.Wrap(err, "查找原创声明卡片失败")
	}

	for _, card := range switchCards {
		text, _ := card.Text()
		if !strings.Contains(text, "原创声明") {
			continue
		}
		switchElem, err := card.Element("div.d-switch")
		if err != nil {
			continue
		}
		checked, err := switchElem.Eval(`() => {
			const input = this.querySelector('input[type="checkbox"]');
			return input ? input.checked : false;
		}`)
		if err == nil && checked.Value.Bool() {
			return nil
		}
		if err := switchElem.Click(proto.InputMouseButtonLeft, 1); err != nil {
			return errors.Wrap(err, "点击原创声明开关失败")
		}
		time.Sleep(500 * time.Millisecond)
		if err := confirmOriginalDeclaration(page); err != nil {
			return err
		}
		return nil
	}
	return errors.New("未找到原创声明选项")
}

func confirmOriginalDeclaration(page *rod.Page) error {
	time.Sleep(800 * time.Millisecond)

	page.MustEval(`() => {
		const footers = document.querySelectorAll('div.footer');
		for (const footer of footers) {
			if (!footer.textContent.includes('原创声明须知')) continue;
			const cb = footer.querySelector('div.d-checkbox input[type="checkbox"]');
			if (cb && !cb.checked) cb.click();
		}
	}`)

	time.Sleep(500 * time.Millisecond)

	page.MustEval(`() => {
		const footers = document.querySelectorAll('div.footer');
		for (const footer of footers) {
			if (!footer.textContent.includes('声明原创')) continue;
			const btn = footer.querySelector('button.custom-button');
			if (btn && !btn.disabled) btn.click();
		}
	}`)

	time.Sleep(300 * time.Millisecond)
	return nil
}

func bindProducts(page *rod.Page, products []string) error {
	if len(products) == 0 {
		return nil
	}

	// Click "添加商品"
	spans, err := page.Elements("span.d-text")
	if err != nil {
		return errors.Wrap(err, "查找商品按钮文本失败")
	}
	var clicked bool
	for _, span := range spans {
		text, _ := span.Text()
		if strings.TrimSpace(text) != "添加商品" {
			continue
		}
		parent := span
		for i := 0; i < 5; i++ {
			p, err := parent.Parent()
			if err != nil {
				break
			}
			parent = p
			tagName, _ := parent.Eval(`() => this.tagName.toLowerCase()`)
			if tagName != nil && tagName.Value.Str() == "button" {
				parent.Click(proto.InputMouseButtonLeft, 1)
				clicked = true
				break
			}
		}
		if clicked {
			break
		}
	}
	if !clicked {
		return errors.New("未找到添加商品按钮")
	}
	time.Sleep(1 * time.Second)

	// Wait for modal
	var modal *rod.Element
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		m, err := page.Element(".multi-goods-selector-modal")
		if err == nil && m != nil {
			if vis, _ := m.Visible(); vis {
				modal = m
				break
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	if modal == nil {
		return errors.New("等待商品选择弹窗超时")
	}

	// Search and select each product
	for _, keyword := range products {
		searchInput, err := modal.Element(`input[placeholder="搜索商品ID 或 商品名称"]`)
		if err != nil {
			continue
		}
		_ = searchInput.SelectAllText()
		time.Sleep(100 * time.Millisecond)
		_ = searchInput.Input(keyword)
		time.Sleep(300 * time.Millisecond)
		_ = page.Keyboard.Press(input.Enter)
		time.Sleep(1500 * time.Millisecond)

		checkbox, err := modal.Element(".goods-list-normal .good-card-container .d-checkbox")
		if err == nil {
			_ = checkbox.Click(proto.InputMouseButtonLeft, 1)
			randomDelay := 800 + rand.Intn(700)
			time.Sleep(time.Duration(randomDelay) * time.Millisecond)
		}
	}

	// Click save
	btn, err := modal.Element(".goods-selected-footer button")
	if err == nil && btn != nil {
		_ = btn.Click(proto.InputMouseButtonLeft, 1)
	}
	time.Sleep(2 * time.Second)
	return nil
}
