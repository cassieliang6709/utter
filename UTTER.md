# Utter

Utter 是 Voxt（sglang-omni 仓库的 `Voxt/`）的个人构建。它让本机 LLM 也能边说边润色，松开按键后等得更短。

## 和上游 Voxt 的区别

分支 `cassie` 在上游 `main` 上加了这些改动：

1. 本机 LLM 也边说边润色：上游在 LLM 地址是 127.0.0.1 时关掉这个功能。这里加了设置项 `features.transcribe.liveCleanLocal`，默认开。上游 PR：sgl-project/sglang-omni#4643（那边默认关）。
2. 最终文字只差标点时照样复用：松开后的最终文字常比实时文字多一个或少一个「，」。以前从这个标点起已润色的部分全部作废。现在只差标点、空格或大小写时，按实时文字核对。历史记录里的识别原文仍是最终文字。
3. 词典超过 30 条也边说边润色：每块请求只带这块文字提到的词条，和松开后整段润色给这几句的词条相同。
4. 改名：app 叫 Utter，bundle id 是 `dev.cassie.utter`，可以和官方 Voxt 并存。数据目录仍是 `~/Library/Application Support/VoxtNew`，词典、历史和设置共用。界面里的文字仍写 Voxt。

## 实测

机器：M1 Pro 16 GB。语音识别：Qwen3-ASR 0.6B 4bit。润色：llama.cpp，Qwen3-4B Q4_0，127.0.0.1:8090。

方法：5 段 7–39 秒的中文录音，由 helper 回放代替麦克风，音频结束 0.5 秒后松开。每段改前、改后交替各 3 次。计时从松开到文字插入。

| 录音 | 时长 | 改前 | 改后 |
|---|---|---|---|
| A | 7.4 s | 0.80 s | 0.28 s |
| B | 12.8 s | 1.01 s | 1.44 s |
| C | 16.5 s | 1.31 s | 0.45 s |
| D | 23.9 s | 1.60 s | 0.57 s |
| E | 38.7 s | 2.77 s | 2.78 s |
| 全部中位数 | | 1.31 s | 0.57 s |

1. 全部复用：15 次里有 9 次松开后不再请求 LLM。只加第 1 项改动时是 5 次。
2. B 没变快：最终文字把「嗯」识别成了「呃」。字不同，按设计重新润色。
3. E 没变快：后半段有字不同，只复用了前 43 个字。其中一次 LLM 花了 8.7 秒。
4. 代价：边录音边润色和语音识别共用 GPU，最终文字慢约 40 ms，第一段字幕晚约 60 ms，字幕更新间隔不变（1.01 s）。

测量时词典没有发给 LLM（glossary 关闭），以便和前几轮对比。

## 编译和安装

需要 Go 1.27.4 和 Xcode。命令在 `Voxt/` 下执行：

```sh
scripts/build-helper.sh
cp -R /Applications/Voxt.app/Contents/Resources/omni resources/darwin-arm64/omni
go tool mygo build -skip-dmg
ditto build/darwin-arm64/Utter.app /Applications/Utter.app
```

1. omni 运行时直接从已装的官方 Voxt 复制，省掉约 1 GB 的编译。
2. app 是 ad hoc 签名。第一次打开要在系统设置里给 Utter 麦克风和辅助功能权限。
3. 官方 Voxt 和 Utter 用同一套快捷键，同一时间只开一个。

## 同步上游

```sh
git fetch upstream
git switch cassie
git rebase upstream/main
```

rebase 后跑 `scripts/check.sh` 和 `CGO_ENABLED=0 go test ./internal/session/ ./internal/settings/`。
