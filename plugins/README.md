# プラグイン

`plugins/<フォルダー>/plugin.json` と JS / CSS を置くと、XBrowser のページに機能を追加できます。
同梱の `sns-hub` をこのフォルダーごと `%LocalAppData%\XBrowser\plugins` へコピーすると使えます。
(設定 →「プラグイン」→「プラグインフォルダーを開く」でフォルダーが開きます)

```json
{
  "id": "my-plugin",
  "name": "My Plugin",
  "version": "1.0.0",
  "matches": ["x.com"],
  "allowHosts": [],
  "js": ["main.js"],
  "css": ["style.css"],
  "config": {}
}
```

- `matches`: 実行するホスト(空ならアプリ内で開けるすべてのサイト)
- `allowHosts`: アプリ内のタブで開くサイトに加えるホスト(公開サフィックス・IPは不可)
- `js` / `css`: フォルダー内の相対パス(各512KB、合計2MBまで)

スクリプトでは `xb` が使えます: `xb.config` / `xb.css(text)` / `xb.onReady(fn)` / `xb.storage.get|set|remove` /
`xb.openTab(url)` / `xb.openExternal(url)` / `xb.isAllowedHost(host)` / `xb.host` / `xb.lang` / `xb.isX`。

プラグインは対象ページ内で自由にスクリプトを実行できます。信頼できるものだけを入れてください。
