//go:build windows

package main

const (
	layoutTabsTop    = "tabs-top"
	layoutToolbarTop = "toolbar-top"
	minBarScale      = 80
	maxBarScale      = 140
)

var (
	barScale int32 = 100

	titleBarH, toolbarH, chromeH   int32
	tabsTop, toolbarTop            int32
	themeBtnW, navBtnW, newTabBtnW int32
	tabMinW, tabMaxW, tabGap       int32
	winBtnW, zoomBtnW, zoomLabelW  int32
	zoomClusterW, settingsBtnW     int32
)

func init() { applyBarMetrics() }

func sc(n int32) int32 { return (n*barScale + 50) / 100 }

func clampBarScale(n int) int {
	if n < minBarScale {
		return minBarScale
	}
	if n > maxBarScale {
		return maxBarScale
	}
	return n
}

func applyBarMetrics() {
	barScale = 100
	if cfg.BarScale != 0 {
		barScale = int32(clampBarScale(cfg.BarScale))
	}
	titleBarH, toolbarH = sc(46), sc(44)
	chromeH = titleBarH + toolbarH
	themeBtnW, navBtnW, newTabBtnW = sc(38), sc(38), sc(38)
	tabMinW, tabMaxW, tabGap = sc(96), sc(240), sc(4)
	winBtnW, zoomBtnW, zoomLabelW = sc(48), sc(30), sc(56)
	zoomClusterW = zoomBtnW*2 + zoomLabelW
	settingsBtnW = sc(38)
	if cfg.BarLayout == layoutToolbarTop {
		toolbarTop, tabsTop = 0, toolbarH
	} else {
		tabsTop, toolbarTop = 0, titleBarH
	}
}

func refreshBarLayout() {
	applyBarMetrics()
	if addrEdit != 0 {
		updateAddrFont()
	}
	if app != nil {
		app.layout()
		invalidateBar()
	}
}
