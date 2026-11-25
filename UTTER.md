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

| 录音 | 时长 | 原生默认（4B `on=false`） | Utter + 4B（`on=true`） | Utter + 0.6B `Q8_0`（`on=true`） | 相对原生降幅 |
|---|---|---|---|---|---|
| A | 7.4 s | 0.80 s | 0.28 s | 0.31 s | -61.3% |
| B | 12.8 s | 1.01 s | 1.44 s | 0.48 s | -52.5% |
| C | 16.5 s | 1.31 s | 0.45 s | 1.24 s（两次：1.96 / 0.52 s） | -5.3% |
| D | 23.9 s | 1.60 s | 0.57 s | 0.64 s | -60.0% |
| E | 38.7 s | 2.77 s（最高 4.66 s） | 2.78 s（含 8.7 s 尖刺） | 1.34 s（零尖刺） | -51.6% |
| 全部中位数 | | 1.31 s | 0.57 s | **0.55 s** | **-58.0%** |
| 全部均值 | | 1.64 s | 1.59 s | **0.80 s** | **-51.0%** |

1. 全部复用：4B 改后 15 次里有 9 次松键后不再请求 LLM。只加第 1 项改动时是 5 次。
2. 4B 的短板与 0.6B 补齐：4B 下 B 因最终文字把「嗯」识别成「呃」重跑（1.44 s），E 因后半段改字只复用前 43 字且出现 8.7 s 长尾；换自训 610 MB `Qwen3-0.6B Q8_0 + ngram-simple` 后，B 压至 0.48 s（3.0×），E 压至 1.34 s（零长尾），全量中位 0.55 s、均值 0.80 s，相对 SGLang-Omni 原生默认方案延迟均降超一半。
3. 代价：边录音边润色和语音识别共用 GPU，最终文字慢约 40 ms，第一段字幕晚约 60 ms，字幕更新间隔不变（1.01 s）。

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
