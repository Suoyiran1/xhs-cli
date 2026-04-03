package xiaohongshu

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Suoyiran1/xhs-cli/internal/errors"
	"github.com/avast/retry-go/v4"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/sirupsen/logrus"
)

const (
	defaultMaxAttempts     = 500
	stagnantLimit          = 20
	minScrollDelta         = 10
	maxClickPerRound       = 3
	stagnantCheckThreshold = 2
	largeScrollTrigger     = 5
	buttonClickInterval    = 3
	finalSprintPushCount   = 15
)

type delayConfig struct {
	min, max int
}

var (
	humanDelayRange   = delayConfig{300, 700}
	reactionTimeRange = delayConfig{300, 800}
	hoverTimeRange    = delayConfig{100, 300}
	readTimeRange     = delayConfig{500, 1200}
	shortReadRange    = delayConfig{600, 1200}
	scrollWaitRange   = delayConfig{100, 200}
	postScrollRange   = delayConfig{300, 500}
)

type CommentLoadConfig struct {
	ClickMoreReplies    bool
	MaxRepliesThreshold int
	MaxCommentItems     int
	ScrollSpeed         string
}

func DefaultCommentLoadConfig() CommentLoadConfig {
	return CommentLoadConfig{
		ClickMoreReplies:    false,
		MaxRepliesThreshold: 10,
		MaxCommentItems:     0,
		ScrollSpeed:         "normal",
	}
}

type FeedDetailAction struct {
	page *rod.Page
}

func NewFeedDetailAction(page *rod.Page) *FeedDetailAction {
	return &FeedDetailAction{page: page}
}

func (f *FeedDetailAction) GetFeedDetail(ctx context.Context, feedID, xsecToken string, loadAllComments bool, config CommentLoadConfig) (*FeedDetailResponse, error) {
	return f.GetFeedDetailWithConfig(ctx, feedID, xsecToken, loadAllComments, config)
}

func (f *FeedDetailAction) GetFeedDetailWithConfig(ctx context.Context, feedID, xsecToken string, loadAllComments bool, config CommentLoadConfig) (*FeedDetailResponse, error) {
	page := f.page.Context(ctx).Timeout(10 * time.Minute)
	url := makeFeedDetailURL(feedID, xsecToken)

	logrus.Infof("打开 feed 详情页: %s", url)

	err := retry.Do(
		func() error {
			page.MustNavigate(url)
			page.MustWaitDOMStable()
			return nil
		},
		retry.Attempts(3),
		retry.Delay(500*time.Millisecond),
		retry.MaxJitter(1000*time.Millisecond),
	)
	if err != nil {
		return nil, err
	}
	sleepRandom(1000, 1000)

	if err := checkPageAccessible(page); err != nil {
		return nil, err
	}

	if loadAllComments {
		if err := f.loadAllCommentsWithConfig(page, config); err != nil {
			logrus.Warnf("加载全部评论失败: %v", err)
		}
	}

	return f.extractFeedDetail(page, feedID)
}

// commentLoader and loading logic

type commentLoader struct {
	page   *rod.Page
	config CommentLoadConfig
	stats  *loadStats
	state  *loadState
}

type loadStats struct {
	totalClicked int
	totalSkipped int
	attempts     int
}

type loadState struct {
	lastCount      int
	lastScrollTop  int
	stagnantChecks int
}

func (f *FeedDetailAction) loadAllCommentsWithConfig(page *rod.Page, config CommentLoadConfig) error {
	loader := &commentLoader{
		page:   page,
		config: config,
		stats:  &loadStats{},
		state:  &loadState{},
	}
	return loader.load()
}

func (cl *commentLoader) load() error {
	maxAttempts := cl.calculateMaxAttempts()
	scrollInterval := getScrollInterval(cl.config.ScrollSpeed)

	logrus.Info("开始加载评论...")
	scrollToCommentsArea(cl.page)
	sleepRandom(humanDelayRange.min, humanDelayRange.max)

	if cl.checkNoComments() {
		return nil
	}

	for cl.stats.attempts = 0; cl.stats.attempts < maxAttempts; cl.stats.attempts++ {
		if cl.checkComplete() {
			return nil
		}
		if cl.shouldClickButtons() {
			cl.clickButtonsWithRetry()
		}

		currentCount := getCommentCount(cl.page)
		cl.updateState(currentCount)

		if cl.shouldStopAtTarget(currentCount) {
			return nil
		}

		cl.performScroll()
		cl.handleStagnation()

		time.Sleep(scrollInterval)
	}

	cl.performFinalSprint()
	return nil
}

func (cl *commentLoader) calculateMaxAttempts() int {
	if cl.config.MaxCommentItems > 0 {
		return cl.config.MaxCommentItems * 3
	}
	return defaultMaxAttempts
}

func (cl *commentLoader) checkNoComments() bool {
	if checkNoCommentsArea(cl.page) {
		logrus.Infof("检测到无评论区域，跳过加载")
		return true
	}
	return false
}

func (cl *commentLoader) checkComplete() bool {
	if checkEndContainer(cl.page) {
		currentCount := getCommentCount(cl.page)
		sleepRandom(humanDelayRange.min, humanDelayRange.max)
		logrus.Infof("加载完成: %d 条评论", currentCount)
		return true
	}
	return false
}

func (cl *commentLoader) shouldClickButtons() bool {
	return cl.config.ClickMoreReplies && cl.stats.attempts%buttonClickInterval == 0
}

func (cl *commentLoader) clickButtonsWithRetry() {
	clicked, skipped := clickShowMoreButtonsSmart(cl.page, cl.config.MaxRepliesThreshold)
	if clicked > 0 || skipped > 0 {
		cl.stats.totalClicked += clicked
		cl.stats.totalSkipped += skipped
		sleepRandom(readTimeRange.min, readTimeRange.max)

		clicked2, skipped2 := clickShowMoreButtonsSmart(cl.page, cl.config.MaxRepliesThreshold)
		if clicked2 > 0 || skipped2 > 0 {
			cl.stats.totalClicked += clicked2
			cl.stats.totalSkipped += skipped2
			sleepRandom(shortReadRange.min, shortReadRange.max)
		}
	}
}

func (cl *commentLoader) updateState(currentCount int) {
	if currentCount != cl.state.lastCount {
		cl.state.lastCount = currentCount
		cl.state.stagnantChecks = 0
	} else {
		cl.state.stagnantChecks++
	}
}

func (cl *commentLoader) shouldStopAtTarget(currentCount int) bool {
	if cl.config.MaxCommentItems <= 0 {
		return false
	}
	if currentCount >= cl.config.MaxCommentItems {
		logrus.Infof("已达到目标评论数: %d/%d", currentCount, cl.config.MaxCommentItems)
		return true
	}
	return false
}

func (cl *commentLoader) performScroll() {
	currentCount := getCommentCount(cl.page)
	if currentCount > 0 {
		scrollToLastComment(cl.page)
		sleepRandom(postScrollRange.min, postScrollRange.max)
	}

	largeMode := cl.state.stagnantChecks >= largeScrollTrigger
	pushCount := 1
	if largeMode {
		pushCount = 3 + rand.Intn(3)
	}

	_, scrollDelta, currentScrollTop := humanScroll(cl.page, cl.config.ScrollSpeed, largeMode, pushCount)

	if scrollDelta < minScrollDelta || currentScrollTop == cl.state.lastScrollTop {
		cl.state.stagnantChecks++
	} else {
		cl.state.stagnantChecks = 0
		cl.state.lastScrollTop = currentScrollTop
	}
}

func (cl *commentLoader) handleStagnation() {
	if cl.state.stagnantChecks >= stagnantLimit {
		humanScroll(cl.page, cl.config.ScrollSpeed, true, 10)
		cl.state.stagnantChecks = 0
	}
}

func (cl *commentLoader) performFinalSprint() {
	humanScroll(cl.page, cl.config.ScrollSpeed, true, finalSprintPushCount)
}

// Utility functions

func sleepRandom(minMs, maxMs int) {
	if maxMs <= minMs {
		time.Sleep(time.Duration(minMs) * time.Millisecond)
		return
	}
	delay := time.Duration(minMs+rand.Intn(maxMs-minMs)) * time.Millisecond
	time.Sleep(delay)
}

func getScrollInterval(speed string) time.Duration {
	switch speed {
	case "slow":
		return time.Duration(1200+rand.Intn(300)) * time.Millisecond
	case "fast":
		return time.Duration(300+rand.Intn(100)) * time.Millisecond
	default:
		return time.Duration(600+rand.Intn(200)) * time.Millisecond
	}
}

// Button clicking

func clickShowMoreButtonsSmart(page *rod.Page, maxRepliesThreshold int) (clicked, skipped int) {
	elements, err := page.Elements(".show-more")
	if err != nil {
		return 0, 0
	}

	replyCountRegex := regexp.MustCompile(`展开\s*(\d+)\s*条回复`)
	maxClick := maxClickPerRound + rand.Intn(maxClickPerRound)
	clickedInRound := 0

	for _, el := range elements {
		if clickedInRound >= maxClick {
			break
		}
		if !isElementClickable(el) {
			continue
		}
		text, err := el.Text()
		if err != nil {
			continue
		}
		if shouldSkipButton(text, maxRepliesThreshold, replyCountRegex) {
			skipped++
			continue
		}
		if clickElementWithHumanBehavior(page, el, text) {
			clicked++
			clickedInRound++
		}
	}
	return clicked, skipped
}

func isElementClickable(el *rod.Element) bool {
	visible, err := el.Visible()
	if err != nil || !visible {
		return false
	}
	box, err := el.Shape()
	return err == nil && len(box.Quads) > 0
}

func shouldSkipButton(text string, threshold int, regex *regexp.Regexp) bool {
	if threshold <= 0 {
		return false
	}
	matches := regex.FindStringSubmatch(text)
	if len(matches) > 1 {
		if replyCount, err := strconv.Atoi(matches[1]); err == nil && replyCount > threshold {
			return true
		}
	}
	return false
}

func clickElementWithHumanBehavior(page *rod.Page, el *rod.Element, text string) bool {
	var clickSuccess bool

	err := retry.Do(
		func() error {
			el.MustEval(`() => {
				try {
					this.scrollIntoView({behavior: 'smooth', block: 'center'});
				} catch (e) {}
			}`)
			sleepRandom(reactionTimeRange.min, reactionTimeRange.max)

			if box, err := el.Shape(); err == nil && len(box.Quads) > 0 {
				x := float64(box.Quads[0][0]+box.Quads[0][4]) / 2
				y := float64(box.Quads[0][1]+box.Quads[0][5]) / 2
				page.Mouse.MustMoveTo(x, y)
				sleepRandom(hoverTimeRange.min, hoverTimeRange.max)
			}

			if err := el.Click(proto.InputMouseButtonLeft, 1); err != nil {
				return err
			}
			sleepRandom(readTimeRange.min, readTimeRange.max)
			clickSuccess = true
			return nil
		},
		retry.Attempts(3),
		retry.Delay(100*time.Millisecond),
		retry.MaxJitter(200*time.Millisecond),
	)

	if err != nil {
		logrus.Debugf("点击失败 '%s': %v", text, err)
		return false
	}
	return clickSuccess
}

// Scrolling

func humanScroll(page *rod.Page, speed string, largeMode bool, pushCount int) (bool, int, int) {
	beforeTop := getScrollTop(page)
	viewportHeight := page.MustEval(`() => window.innerHeight`).Int()

	baseRatio := getScrollRatio(speed)
	if largeMode {
		baseRatio *= 2.0
	}

	scrolled := false
	actualDelta := 0
	currentScrollTop := beforeTop

	for i := 0; i < max(1, pushCount); i++ {
		scrollDelta := calculateScrollDelta(viewportHeight, baseRatio)
		page.MustEval(`(delta) => { window.scrollBy(0, delta); }`, scrollDelta)
		sleepRandom(scrollWaitRange.min, scrollWaitRange.max)

		currentScrollTop = getScrollTop(page)
		deltaThisTime := currentScrollTop - beforeTop
		actualDelta += deltaThisTime

		if deltaThisTime > 5 {
			scrolled = true
		}
		beforeTop = currentScrollTop

		if i < pushCount-1 {
			sleepRandom(humanDelayRange.min, humanDelayRange.max)
		}
	}

	if !scrolled && pushCount > 0 {
		page.MustEval(`() => window.scrollTo(0, document.body.scrollHeight)`)
		sleepRandom(postScrollRange.min, postScrollRange.max)
		currentScrollTop = getScrollTop(page)
		actualDelta = currentScrollTop - beforeTop + actualDelta
		scrolled = actualDelta > 5
	}

	return scrolled, actualDelta, currentScrollTop
}

func getScrollRatio(speed string) float64 {
	switch speed {
	case "slow":
		return 0.5
	case "fast":
		return 0.9
	default:
		return 0.7
	}
}

func calculateScrollDelta(viewportHeight int, baseRatio float64) float64 {
	scrollDelta := float64(viewportHeight) * (baseRatio + rand.Float64()*0.2)
	if scrollDelta < 400 {
		scrollDelta = 400
	}
	return scrollDelta + float64(rand.Intn(100)-50)
}

func scrollToCommentsArea(page *rod.Page) {
	if el, err := page.Timeout(2 * time.Second).Element(".comments-container"); err == nil {
		el.MustScrollIntoView()
	}
	time.Sleep(500 * time.Millisecond)
	smartScroll(page, 100)
}

func smartScroll(page *rod.Page, delta float64) {
	page.MustEval(`(delta) => {
		let targetElement = document.querySelector('.note-scroller')
			|| document.querySelector('.interaction-container')
			|| document.documentElement;
		const wheelEvent = new WheelEvent('wheel', {
			deltaY: delta,
			deltaMode: 0,
			bubbles: true,
			cancelable: true,
			view: window
		});
		targetElement.dispatchEvent(wheelEvent);
	}`, delta)
}

func scrollToLastComment(page *rod.Page) {
	elements, err := page.Timeout(2 * time.Second).Elements(".parent-comment")
	if err != nil || len(elements) == 0 {
		return
	}
	lastComment := elements[len(elements)-1]
	lastComment.MustScrollIntoView()
}

// DOM queries

func getScrollTop(page *rod.Page) int {
	var result int
	_ = retry.Do(
		func() error {
			evalResult := page.MustEval(`() => {
				return window.pageYOffset || document.documentElement.scrollTop || document.body.scrollTop || 0;
			}`)
			result = evalResult.Int()
			return nil
		},
		retry.Attempts(3),
		retry.Delay(100*time.Millisecond),
	)
	return result
}

func getCommentCount(page *rod.Page) int {
	var result int
	_ = retry.Do(
		func() error {
			elements, err := page.Timeout(2 * time.Second).Elements(".parent-comment")
			if err != nil {
				return err
			}
			result = len(elements)
			return nil
		},
		retry.Attempts(3),
		retry.Delay(100*time.Millisecond),
	)
	return result
}

func getTotalCommentCount(page *rod.Page) int {
	var result int
	_ = retry.Do(
		func() error {
			totalEl, err := page.Timeout(2 * time.Second).Element(".comments-container .total")
			if err != nil {
				return err
			}
			text, err := totalEl.Text()
			if err != nil {
				return err
			}
			re := regexp.MustCompile(`共(\d+)条评论`)
			matches := re.FindStringSubmatch(text)
			if len(matches) > 1 {
				count, err := strconv.Atoi(matches[1])
				if err != nil {
					return err
				}
				result = count
			}
			return nil
		},
		retry.Attempts(3),
		retry.Delay(100*time.Millisecond),
	)
	return result
}

func checkNoCommentsArea(page *rod.Page) bool {
	noCommentsEl, err := page.Timeout(2 * time.Second).Element(".no-comments-text")
	if err != nil {
		return false
	}
	text, err := noCommentsEl.Text()
	if err != nil {
		return false
	}
	return strings.Contains(strings.TrimSpace(text), "这是一片荒地")
}

func checkEndContainer(page *rod.Page) bool {
	var result bool
	_ = retry.Do(
		func() error {
			endEl, err := page.Timeout(2 * time.Second).Element(".end-container")
			if err != nil {
				result = false
				return nil
			}
			text, err := endEl.Text()
			if err != nil {
				result = false
				return nil
			}
			textUpper := strings.ToUpper(strings.TrimSpace(text))
			result = strings.Contains(textUpper, "THE END") || strings.Contains(textUpper, "THEEND")
			return nil
		},
		retry.Attempts(3),
		retry.Delay(100*time.Millisecond),
	)
	return result
}

// Page check

func checkPageAccessible(page *rod.Page) error {
	time.Sleep(500 * time.Millisecond)

	wrapperEl, err := page.Timeout(2 * time.Second).Element(".access-wrapper, .error-wrapper, .not-found-wrapper, .blocked-wrapper")
	if err != nil {
		return nil
	}

	text, err := wrapperEl.Text()
	if err != nil {
		return nil
	}

	keywords := []string{
		"当前笔记暂时无法浏览", "该内容因违规已被删除", "该笔记已被删除",
		"内容不存在", "笔记不存在", "已失效", "私密笔记", "仅作者可见",
		"因用户设置，你无法查看", "因违规无法查看",
	}

	for _, kw := range keywords {
		if strings.Contains(text, kw) {
			return fmt.Errorf("笔记不可访问: %s", kw)
		}
	}

	trimmedText := strings.TrimSpace(text)
	if trimmedText != "" {
		return fmt.Errorf("笔记不可访问: %s", trimmedText)
	}

	return nil
}

// Data extraction

func (f *FeedDetailAction) extractFeedDetail(page *rod.Page, feedID string) (*FeedDetailResponse, error) {
	var result string

	err := retry.Do(
		func() error {
			evalResult := page.MustEval(`() => {
				if (window.__INITIAL_STATE__ &&
					window.__INITIAL_STATE__.note &&
					window.__INITIAL_STATE__.note.noteDetailMap) {
					return JSON.stringify(window.__INITIAL_STATE__.note.noteDetailMap);
				}
				return "";
			}`).String()

			if evalResult != "" {
				result = evalResult
				return nil
			}
			return fmt.Errorf("无法获取初始状态数据")
		},
		retry.Attempts(3),
		retry.Delay(200*time.Millisecond),
	)

	if err != nil {
		return nil, fmt.Errorf("提取Feed详情失败: %w", err)
	}

	if result == "" {
		return nil, errors.ErrNoFeedDetail
	}

	var noteDetailMap map[string]struct {
		Note     FeedDetail  `json:"note"`
		Comments CommentList `json:"comments"`
	}

	if err := json.Unmarshal([]byte(result), &noteDetailMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal noteDetailMap: %w", err)
	}

	noteDetail, exists := noteDetailMap[feedID]
	if !exists {
		return nil, fmt.Errorf("feed %s not found in noteDetailMap", feedID)
	}

	return &FeedDetailResponse{
		Note:     noteDetail.Note,
		Comments: noteDetail.Comments,
	}, nil
}

func makeFeedDetailURL(feedID, xsecToken string) string {
	return fmt.Sprintf("https://www.xiaohongshu.com/explore/%s?xsec_token=%s&xsec_source=pc_feed", feedID, xsecToken)
}
