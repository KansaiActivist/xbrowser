(function(){
var PAGES=[
["概要",[["index.html","ホーム"],["features.html","機能一覧"]]],
["使い方",[["usage.html","基本操作とショートカット"],["settings.html","設定画面"],["profiles.html","プロファイル"]]],
["開発者向け",[["build.html","ビルド方法"],["architecture.html","仕組み・構成"],["config.html","config.json リファレンス"]]],
["サポート",[["troubleshooting.html","トラブルシューティング"]]]
];
var cur=location.pathname.split("/").pop()||"index.html";
var flat=[];PAGES.forEach(function(g){g[1].forEach(function(p){flat.push(p)})});
var h=document.createElement("header");
h.innerHTML='<button id="menu" aria-label="menu">☰</button><a class="brand" href="index.html">xbrowser Wiki</a><span class="sp"></span><button id="theme">🌓</button>';
document.body.prepend(h);
var nav=document.getElementById("nav");
var html='<input type="search" placeholder="ページを検索" id="q">';
PAGES.forEach(function(g){html+='<h4>'+g[0]+'</h4>';g[1].forEach(function(p){html+='<a href="'+p[0]+'"'+(p[0]===cur?' class="on"':'')+'>'+p[1]+'</a>'})});
nav.innerHTML=html;
document.getElementById("q").addEventListener("input",function(e){
var v=e.target.value.toLowerCase();
nav.querySelectorAll("a").forEach(function(a){a.style.display=a.textContent.toLowerCase().indexOf(v)<0?"none":""});
});
document.getElementById("menu").onclick=function(){nav.classList.toggle("open")};
var root=document.documentElement;
try{var s=localStorage.getItem("theme");if(s)root.setAttribute("data-theme",s)}catch(e){}
document.getElementById("theme").onclick=function(){
var dark=root.getAttribute("data-theme")==="dark"||(!root.getAttribute("data-theme")&&matchMedia("(prefers-color-scheme:dark)").matches);
var n=dark?"light":"dark";root.setAttribute("data-theme",n);
try{localStorage.setItem("theme",n)}catch(e){}
};
var i=flat.findIndex(function(p){return p[0]===cur});
var pg=document.getElementById("pager");
if(pg&&i>=0){
pg.innerHTML=(i>0?'<a href="'+flat[i-1][0]+'">← '+flat[i-1][1]+'</a>':'<span></span>')+(i<flat.length-1?'<a href="'+flat[i+1][0]+'">'+flat[i+1][1]+' →</a>':'<span></span>');
}
})();
