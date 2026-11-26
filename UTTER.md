# Utter

Utter 是 Voxt（sglang-omni 仓库的 `Voxt/`）的个人构建。语音识别、润色都在这台 Mac 上跑，松开按键后约 0.44 秒出字（中位数）。

## 和上游 Voxt 的区别

分支 `cassie` 在上游 `main` 上加了这些改动：

1. 本机 LLM 也边说边润色：上游在 LLM 地址是 127.0.0.1 时关掉这个功能。这里加了设置项 `features.transcribe.liveCleanLocal`，默认开。上游 PR：sgl-project/sglang-omni#4643（那边默认关）。
2. 最终文字只差标点时照样复用：松开后的最终文字常比实时文字多一个或少一个「，」，以前从这个标点起已润色的部分全部作废。现在只差标点、空格或大小写时，按实时文字核对。历史记录里的识别原文仍是最终文字。
3. 词典超过 30 条也边说边润色：每块请求只带这块文字提到的词条，和松开后整段润色给这几句的词条相同。
4. 长录音更早切段：`asr.fast` 从 20 秒起（上游 60 秒）在停顿处定下已识别的片段，片段边录边润色，松开后只剩最后一段。
5. 松开按键时本机 LLM 让出 GPU：润色模型在本机时，松开后暂停新的润色请求、取消没被采纳的在途请求，让语音识别的最终一遍先跑。已经润色好的块照常复用。
6. 精简界面（编译标签 `utter`）：侧边栏只留首页、历史、转录、词典、日志。翻译、改写、笔记、会议、按应用增强、快捷短语连同快捷键一起隐藏。语音模型只列 Qwen3-ASR 0.6B 4bit。更新检查和引导隐藏。设置里的「语言模型」保留，本机润色服务的地址在那里改。
7. 改名：app 叫 Utter，bundle id 是 `dev.cassie.utter`，可以和官方 Voxt 并存。数据目录仍是 `~/Library/Application Support/VoxtNew`，词典、历史和设置共用。界面文字仍写 Voxt。

源码里还有一处没进 app 的改动：`sglang_omni_mlx/native` 里 Qwen3-ASR 的 C++ 缓存（位置编码、提示词 token）。app 打包的 omni 运行时从官方 Voxt 2.1.3 复制，和官方逐字节相同，这部分 C++ 没有编译进去，也没有测过。

## 实测

机器：M1 Pro 16 GB。语音识别：Qwen3-ASR 0.6B 4bit。方法：5 段 7–39 秒的中文录音由 helper 回放代替麦克风，音频结束 0.5 秒后松开；每段开关关、开交替各 3 次。计时从松开到文字插入，表里是中位数。测量时词典没有发给 LLM。

| 录音 | 时长 | 4B，关 | 4B，开 | 0.6B run3，关 | 0.6B run3，开 | 0.6B run4，关 | 0.6B run4，开 |
|---|---|---|---|---|---|---|---|
| A | 7.4 s | 0.80 s | 0.28 s | 0.49 s | 0.32 s | 0.42 s | 0.28 s |
| B | 12.8 s | 1.01 s | 1.44 s | 0.53 s | 0.56 s | 0.48 s | 0.49 s |
| C | 16.5 s | 1.31 s | 0.45 s | 0.70 s | 0.54 s | 0.63 s | 0.44 s |
| D | 23.9 s | 1.60 s | 0.57 s | 0.24 s | 0.26 s | 0.25 s | 0.22 s |
| E | 38.7 s | 2.77 s | 2.78 s | 0.55 s | 0.62 s | 0.54 s | 0.51 s |
| 全部 | | 1.31 s | 0.57 s | 0.53 s | 0.54 s | 0.48 s | 0.44 s |

- 4B：llama.cpp，Qwen3-4B-Instruct Q4_0，代码到第 3 项为止。
- 0.6B：`voxt_polish_cpp`，自训 Qwen3-0.6B Q8_0（见 `voxt-polish/`），代码到第 6 项。run3 是第三轮训练（324 条清洗语料 + 24 条合成锚点）；run4 再补 26 条句中改口锚点（共 50 条），10 条没见过的改口测试从 1/10 升到 7/10（Mac 上 Q8_0 复测，结果同 4090 bf16）。8090 现在跑 run4。

1. 4B 上边说边润色有用：中位数 1.31 → 0.57 秒，15 次里 9 次松开后不再请求 LLM。B 的「嗯」被最终文字识别成「呃」，字不同，所以重新润色。
2. 0.6B 上开关的差别变小：润色一段只要零点几秒，没剩多少可以提前做（run3 0.53 / 0.54 秒，run4 0.48 / 0.44 秒）。从 4B 换到 0.6B 才是这一步的主要收益（关着时 1.31 → 0.48 秒）。
3. D、E 超过 20 秒，走第 4 项的切段润色，开关关着也会边录边润色，所以 0.6B 下 D 关着只要 0.24 秒。
4. 代价：边录边润色和语音识别共用 GPU。4B 下最终识别慢约 40 ms，第一段字幕晚约 60 ms，字幕更新间隔不变（1.01 秒）。0.6B 下第一段字幕和更新间隔都不变（0.35 秒、1.02 秒）。E 开着时有一次 1.01 秒。

## 编译和安装

需要 Go 1.27.4 和 Xcode，在 `Voxt/` 下执行：

```sh
~/Developer/voxt-polish/cpp/build-static.sh   # 润色引擎，一个不依赖 Homebrew 的二进制
scripts/build-helper.sh
scripts/build-utter.sh
```

1. `build-utter.sh` 把润色引擎和训练时的提示词放进 app（`Contents/Resources/polish`），从 `/Applications/Voxt.app` 复制 omni 运行时，删掉精简版用不到的 6 个语音服务端（Whisper、Parakeet、SenseVoice、Cohere、Nemotron、MOSS，共约 8 MB），用 `-tags utter` 编译，装到 `/Applications/Utter.app`。
2. 默认是 ad hoc 签名，每次重编签名都会变，辅助功能权限要重新打开一次。发布版这样编：`UTTER_SIGN_IDENTITY="Developer ID Application: …" UTTER_NOTARY_PROFILE=<notarytool 配置名> UTTER_DMG=1 scripts/build-utter.sh`，签名、公证、打 DMG 一起做，签名身份固定后重编不再丢权限。
3. 官方 Voxt 和 Utter 用同一套快捷键，同一时间只开一个。

## 测试

```sh
scripts/check.sh
CGO_ENABLED=0 go test ./...
CGO_ENABLED=0 go test -tags utter -run TestUtterProfile .
```

不带标签时全部测试检查上游界面。带 `-tags utter` 时只跑 `TestUtterProfile*`：其余界面测试断言的是被隐藏的页面，在精简版里本来就会失败。

## 同步上游

```sh
git fetch upstream
git switch cassie
git rebase upstream/main
```

rebase 后跑上面的测试。
