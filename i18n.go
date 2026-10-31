//go:build windows

package main

import (
	"fmt"
	"syscall"
)

var languages = []string{"ja", "en", "zh", "ko"}

func isLang(s string) bool {
	for _, l := range languages {
		if s == l {
			return true
		}
	}
	return false
}

func detectLanguage() string {
	p := syscall.NewLazyDLL("kernel32.dll").NewProc("GetUserDefaultUILanguage")
	if p.Find() != nil {
		return "en"
	}
	id, _, _ := p.Call()
	switch id & 0x3ff {
	case 0x11:
		return "ja"
	case 0x04:
		return "zh"
	case 0x12:
		return "ko"
	}
	return "en"
}

var goStrings = map[string][4]string{
	"menu_new_tab":     {"新しいタブ", "New tab", "新标签页", "새 탭"},
	"menu_profile_new": {"プロファイルを選んで新しいタブ", "New tab with profile", "选择配置文件新建标签页", "프로필을 골라 새 탭"},
	"profile_default":  {"Default(通常)", "Default (standard)", "Default(默认)", "Default(기본)"},
	"menu_profile_add": {"新しいプロファイルを追加…", "Add new profile…", "添加新配置文件…", "새 프로필 추가…"},
	"menu_close_tab":   {"このタブを閉じる", "Close this tab", "关闭此标签页", "이 탭 닫기"},
	"menu_tab_profile": {"このタブのプロファイル(現在: %s)", "Profile for this tab (current: %s)", "此标签页的配置文件(当前:%s)", "이 탭의 프로필(현재: %s)"},
	"menu_settings":    {"設定", "Settings", "设置", "설정"},
	"menu_start":       {"スタートページを開く", "Open start page", "打开起始页", "시작 페이지 열기"},
	"menu_zoom_reset":  {"拡大率を100%に戻す", "Reset zoom to 100%", "将缩放恢复为 100%", "확대율을 100%로 되돌리기"},

	"ib_title":  {"新しいプロファイル", "New profile", "新配置文件", "새 프로필"},
	"ib_prompt": {"プロファイル名(例: 仕事用、サブ垢1 など):", "Profile name (e.g. Work, Account 2):", "配置文件名称(例如:工作、小号1):", "프로필 이름(예: 업무용, 서브계정1):"},
	"ib_cancel": {"キャンセル", "Cancel", "取消", "취소"},

	"err_webview2": {
		"WebView2 ランタイムの初期化に失敗しました。\r\nMicrosoft Edge WebView2 Runtime をインストールしてください。",
		"Failed to initialize the WebView2 runtime.\r\nPlease install the Microsoft Edge WebView2 Runtime.",
		"WebView2 运行时初始化失败。\r\n请安装 Microsoft Edge WebView2 Runtime。",
		"WebView2 런타임을 초기화하지 못했습니다.\r\nMicrosoft Edge WebView2 Runtime을 설치해 주세요.",
	},

	"title_new":      {"新しいタブ", "New tab", "新标签页", "새 탭"},
	"title_settings": {"設定", "Settings", "设置", "설정"},
	"title_start":    {"スタートページ", "Start page", "起始页", "시작 페이지"},

	"toast_exists":   {"「%s」は既にあります", "\"%s\" already exists", "“%s”已存在", "\"%s\"은(는) 이미 있습니다"},
	"toast_toolong":  {"プロファイル名は30文字以内にしてください", "Profile names must be 30 characters or fewer", "配置文件名称不能超过 30 个字符", "프로필 이름은 30자 이내여야 합니다"},
	"toast_added":    {"プロファイル「%s」を追加しました", "Added profile \"%s\"", "已添加配置文件“%s”", "프로필 \"%s\"을(를) 추가했습니다"},
	"toast_renamed":  {"プロファイル名を「%s」に変更しました", "Renamed profile to \"%s\"", "已将配置文件重命名为“%s”", "프로필 이름을 \"%s\"(으)로 변경했습니다"},
	"toast_bad_name": {"プロファイル名を入力してください", "Please enter a profile name", "请输入配置文件名称", "프로필 이름을 입력해 주세요"},
	"toast_removed":  {"プロファイル「%s」を削除しました", "Removed profile \"%s\"", "已删除配置文件“%s”", "프로필 \"%s\"을(를) 삭제했습니다"},

	"pick_title":    {"背景に使う画像を選択", "Choose a background image", "选择背景图片", "배경으로 사용할 이미지 선택"},
	"pick_filter":   {"画像ファイル", "Image files", "图片文件", "이미지 파일"},
	"toast_img_ok":  {"背景画像を設定しました", "Background image set", "已设置背景图片", "배경 이미지를 설정했습니다"},
	"toast_img_bad": {"画像を読み込めませんでした", "Could not load the image", "无法读取图片", "이미지를 불러오지 못했습니다"},
	"toast_img_big": {"画像が大きすぎます(20MBまで)", "The image is too large (max 20 MB)", "图片过大(最大 20MB)", "이미지가 너무 큽니다(최대 20MB)"},
}

func langIndex() int {
	for i, l := range languages {
		if l == cfg.Language {
			return i
		}
	}
	return 0
}

func T(key string, args ...any) string {
	v, ok := goStrings[key]
	if !ok {
		return key
	}
	s := v[langIndex()]
	if s == "" {
		s = v[1]
	}
	if len(args) > 0 {
		return fmt.Sprintf(s, args...)
	}
	return s
}
