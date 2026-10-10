const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");
const assets = path.resolve(__dirname, "../assets");
const prompt = "Run this cloud installation command: synthetic command";
const labels = { en: "Copy installation prompt", zh: "复制安装提示词", "zh-TW": "複製安裝提示詞", ja: "インストール用プロンプトをコピー", de: "Installationsprompt kopieren" };
const copiedLabels = {en:"Copied — send it to Grok Bot",zh:"已复制，请发给 Grok Bot","zh-TW":"已複製，請傳給 Grok Bot",ja:"コピーしました。Grok Bot に送信してください",de:"Kopiert — an Grok Bot senden"};
for (const engine of ["chromium", "webkit"]) test(engine + ": Grok Bot cloud prompt", async (t) => {
 const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
 t.after(() => browser.close());
 for (const [lang, label] of Object.entries(labels)) await t.test(lang, async () => {
  const context = await browser.newContext({ viewport: {width:980, height:620} });
  const page = await context.newPage(); page.setDefaultTimeout(5000); const copies = []; const errors = [];
  page.on("pageerror", e => errors.push(e.message));
  await page.route("**/*", async route => {
   const req = route.request(), url = new URL(req.url());
   if (url.pathname === "/boot.js") return route.fulfill({ contentType:"text/javascript", body: `window.bootPrefs = {lang:${JSON.stringify(lang)},theme:"light",web:true};` });
   if (url.pathname === "/wails/runtime.js") return route.fulfill({ contentType:"text/javascript", body:"export const Window = {};" });
   if (url.pathname === "/api/state") return route.fulfill({json:{agents:[{id:"grok-bot",name:"Grok Bot",wired:true,path:"/test/grok-bot",launch:prompt,fields:[{key:"model",label:"model",value:"magpie/vendor/model",options:[{value:"magpie/vendor/model",label:"magpie/vendor/model",ref:"vendor/model"}]},{key:"gateway-url",label:"Gateway URL",value:"https://gateway.example/v1",options:[]}]}],profiles:[],settings:{lang,theme:"light"}}});
   if (url.pathname === "/api/copy") {copies.push(req.postDataJSON().text);return route.fulfill({json:{}});}
   if (url.pathname === "/api/usage/quotas") return route.fulfill({json:[]});
   if (url.pathname === "/api/groups") return route.fulfill({json:{groups:[]}});
   if (url.pathname === "/api/providers") return route.fulfill({json:{providers:[],gateway:{running:true}}});
   if (url.pathname.startsWith("/api/")) return route.fulfill({json:{}});
   const file = path.join(assets, url.pathname === "/" ? "index.html" : url.pathname);
   await route.fulfill({body:await fs.readFile(file),contentType:{".html":"text/html",".js":"text/javascript",".css":"text/css",".svg":"image/svg+xml",".png":"image/png"}[path.extname(file)]});
  });
  await page.goto("http://magpie.test/?view=agents");
  assert.deepEqual(errors, []);
  const row = page.locator('.row.agent[data-id="grok-bot"]'); await row.waitFor();
  const button = row.locator(".field.launch"); assert.ok((await button.getAttribute("title")).startsWith(label));
  const copied = page.waitForResponse(r => new URL(r.url()).pathname === "/api/copy");
  await button.click(); await copied; assert.deepEqual(copies,[prompt]);
  await row.locator(".ag-link").click();
  const detail = row.locator(".ag-exp"); await detail.waitFor();
  const detailCopy = detail.locator("code").filter({hasText:prompt}).locator("..").locator("button.ag-quiet");
  const detailCopied = page.waitForResponse(r => new URL(r.url()).pathname === "/api/copy");
  await detailCopy.click(); await detailCopied;
  await page.getByText(copiedLabels[lang], {exact:true}).waitFor();
  assert.deepEqual(copies,[prompt,prompt]); assert.deepEqual(errors,[]);
  await context.close();
 });
});
