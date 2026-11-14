# XBrowser Share (ブラウザー拡張機能)

他のブラウザーで「Xで共有」「Xでポスト」(`x.com/intent/post` など)を押したとき、
XBrowser の投稿画面を開く拡張機能です。

## 使い方

1. XBrowser の「設定」→「一般」→「他のブラウザーからの共有」をオンにします(`xbrowser://` を登録)。
2. この `extension` フォルダを、使っているブラウザーに読み込みます。
   - **Chrome / Edge**: `chrome://extensions`(Edge は `edge://extensions`)→「デベロッパー モード」をオン →
     「パッケージ化されていない拡張機能を読み込む」→ このフォルダを選ぶ。
   - **Firefox**: `about:debugging` →「このFirefox」→「一時的なアドオンを読み込む」→ `manifest.json` を選ぶ
     (Firefox は再起動すると外れます)。
3. 共有ボタンを押すと、ブラウザーが「XBrowser を開きますか?」と確認します。
   「常に許可」にすると、次からは確認なしで開きます。

## 動作

- 対象は `x.com` / `twitter.com` の `/intent/post`、`/intent/tweet`、`/share` のページだけです。
- `text` `url` `hashtags` `via` `related` `in_reply_to` だけを引き継ぎ、`xbrowser://share?...` に変換します。
- ポップアップやリンク先として開かれた共有用のタブ・ウィンドウは、XBrowser に渡したあとで自動で閉じます。
  通常のタブで開いた場合は閉じません。
- この拡張機能は通信を行いません。必要な権限は `webNavigation`(ページ移動の検知)だけです。
