(function(){
var PAGES=[
["概要",[["index.html","ホーム"],["features.html","機能一覧"]]],
["使い方",[["usage.html","基本操作とショートカット"],["settings.html","設定画面"],["profiles.html","プロファイル"]]],
["開発者向け",[["build.html","ビルド方法"],["architecture.html","仕組み・構成"],["config.html","config.json リファレンス"]]],
["サポート",[["troubleshooting.html","トラブルシューティング"]]]
];
var flat=[];PAGES.forEach(function(g){g[1].forEach(function(p){flat.push(p)})});
function fileOf(path){return path.split("/").pop()||"index.html"}
var cur=fileOf(location.pathname);
var h=document.createElement("header");
h.innerHTML='<button id="menu" aria-label="menu">☰</button><a class="brand" href="index.html">xbrowser Wiki</a><span class="sp"></span><button id="theme">🌓</button>';
document.body.prepend(h);
var nav=document.getElementById("nav");
var article=document.querySelector("article");
var html='<input type="search" placeholder="ページを検索" id="q">';
PAGES.forEach(function(g){html+='<h4>'+g[0]+'</h4>';g[1].forEach(function(p){html+='<a href="'+p[0]+'" data-p="'+p[0]+'">'+p[1]+'</a>'})});
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

function markActive(){
nav.querySelectorAll("a").forEach(function(a){a.classList.toggle("on",a.getAttribute("data-p")===cur)});
}
function buildPager(){
var i=flat.findIndex(function(p){return p[0]===cur});
var pg=document.getElementById("pager");
if(!pg||i<0)return;
pg.innerHTML=(i>0?'<a href="'+flat[i-1][0]+'">← '+flat[i-1][1]+'</a>':'<span></span>')+(i<flat.length-1?'<a href="'+flat[i+1][0]+'">'+flat[i+1][1]+' →</a>':'<span></span>');
}
markActive();buildPager();

var cache={};
cache[cur]={title:document.title,body:article.innerHTML};
var pending={};
function load(file){
if(cache[file])return Promise.resolve(cache[file]);
if(pending[file])return pending[file];
pending[file]=fetch(file).then(function(r){
if(!r.ok)throw new Error(r.status);
return r.text();
}).then(function(t){
var d=new DOMParser().parseFromString(t,"text/html");
var a=d.querySelector("article");
if(!a)throw new Error("no article");
cache[file]={title:d.title,body:a.innerHTML};
return cache[file];
});
pending[file].catch(function(){delete pending[file]});
return pending[file];
}

var token=0;
function show(file,hash,push){
var my=++token;
load(file).then(function(c){
if(my!==token)return;
article.classList.add("swap");
setTimeout(function(){
if(my!==token)return;
article.innerHTML=c.body;
document.title=c.title;
cur=file;
markActive();buildPager();
nav.classList.remove("open");
if(push)history.pushState({f:file},"",file+(hash||""));
var el=hash&&document.getElementById(decodeURIComponent(hash.slice(1)));
if(el)el.scrollIntoView();else window.scrollTo(0,0);
article.classList.remove("swap");
},120);
}).catch(function(){location.href=file+(hash||"")});
}

function isInternal(a){
if(a.target||a.hasAttribute("download"))return false;
var u=new URL(a.href,location.href);
if(u.origin!==location.origin)return false;
var f=fileOf(u.pathname);
return flat.some(function(p){return p[0]===f})&&u.pathname.replace(/[^/]*$/,"")===location.pathname.replace(/[^/]*$/,"");
}
document.addEventListener("click",function(e){
if(e.defaultPrevented||e.button!==0||e.metaKey||e.ctrlKey||e.shiftKey||e.altKey)return;
var a=e.target.closest&&e.target.closest("a[href]");
if(!a||!isInternal(a))return;
var u=new URL(a.href,location.href);
var f=fileOf(u.pathname);
e.preventDefault();
if(f===cur){
var el=u.hash&&document.getElementById(decodeURIComponent(u.hash.slice(1)));
if(el)el.scrollIntoView();else window.scrollTo({top:0,behavior:"smooth"});
return;
}
show(f,u.hash,true);
});
document.addEventListener("mouseover",function(e){
var a=e.target.closest&&e.target.closest("a[href]");
if(a&&isInternal(a))load(fileOf(new URL(a.href,location.href).pathname)).catch(function(){});
});
window.addEventListener("popstate",function(){
var f=fileOf(location.pathname);
if(f!==cur)show(f,location.hash,false);
});
history.replaceState({f:cur},"",location.href);

if(location.protocol!=="file:"){
var idle=window.requestIdleCallback||function(f){setTimeout(f,800)};
idle(function(){flat.forEach(function(p){load(p[0]).catch(function(){})})});
}
})();
