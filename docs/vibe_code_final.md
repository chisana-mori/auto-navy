
# Vibe Code

欢迎来到 Vibe 编码世界，这里是探索 AI 编程辅助、智能体发展、以及各种炫酷工具的聚集地。

---

## 🧠 学习板块

最近我们主打两个 AI 工具：

- **[DeepWiki](https://deepwiki.com/)**：直接把 GitHub 仓库变成可互动的知识百科，还能对接 GPT 和 Gemini。

- **[OpenDeepWiki](https://github.com/opendeepwiki)**：开源版，支持本地部署，RAG 架构，知识问答也不在话下。

我们尝试通过这两个工具去看 kubelet 源码，体验还不错，RAG + 源码 = 爽！
👉 [Kubelet 源码仓库](https://github.com/kubernetes/kubernetes/tree/master/pkg/kubelet)

---

## 🔍 AI Code 的进化史

从早期的 IDE 补全到现在的 AI 智能体，我们走了这么几个阶段：

0. IDE 补全（最早的智力点火）  
1. AI 问答（ChatGPT/Gemini 入门）  
2. 人机结对编程（像和个聪明队友搭档）  
3. AI Agent（智能自动化写代码）

---

## 🛠 工具和插件盘点

### 常用工具：

- **[Cursor](https://www.cursor.so/)**：由 Cursor 团队开发的编程 IDE，支持上下文 AI 辅助。  
  **付费情况**：提供免费试用，Pro 版 $20/月，团队版 $40/用户/月。

- **[Windsurf](https://windsurfai.org/)**：由 Codeium 开发的多模型聚合 + 云端服务工具。  
  **付费情况**：
  - 免费版：提供基本功能。
  - Pro 版：$15/月，包含 500 用户提示积分和 1,500 流程操作积分。
  - Pro Ultimate 版：$60/月，提供无限用户提示积分和 3,000 流程操作积分。
  - 团队版：$30/用户/月。
  - 企业版：定制化价格。

- **[Trae](https://trae.ai/)**：由 Trae Labs 开发的 agent 流水线实验场。  
  **付费情况**：目前提供免费试用，具体定价参考官网。

### 插件精选：

- **[Augment](https://augment.sh/)**：由 Augment 开发的增强型插件。  
  **付费情况**：Dev 版 $50/月。

- **[Copilot](https://github.com/features/copilot)**：由 GitHub 开发的 AI 编程助手。  
  **付费情况**：
  - 个人版：$10/月。
  - 企业版：$19/用户/月。

- **[Roo](https://roo.ai/)**：由 Roo AI 开发的智能插件。  

- **[Cline](https://cline.io/)**：由 Cline 开发，支持对接本地模型。

---

## 🧑‍💻 编程模型推荐

现阶段实测：

- **Claude 3.7 Sonnet**：审查、测试类任务稳如老狗。
- **Gemini 2.5 Pro**：编码 & 文档生成一把好手。

### 模型能力对比：

| 能力领域       | Claude 3.7 Sonnet | Gemini 2.5 Pro |
|----------------|-------------------|----------------|
| 数学推理       | 80%               | 92%            |
| 科学推理       | 79%               | 93%            |
| 代码生成       | 70.3%             | 63.8%          |
| 多步骤推理     | 82%               | 85%            |
| 事实准确性     | 86%               | 83%            |
| 长文本处理     | 200K tokens       | 1M tokens      |
| 多模态处理     | 文本              | 文本、图像、音频 |
| 编程优势       | 调试、文档总结    | 全栈、多语言   |

来源：

- [Cursor Blog: Gemini vs Claude](https://www.cursor-ide.com/blog/gemini-claude-comparison-2025)
- [ChatHub Blog: Model Comparison](https://blog.chathub.gg/gemini-2-5-pro-vs-claude-3-7-sonnet-a-comprehensive-comparison-analysis-of-ai-models/)

---

## 🤖 AI Agent 原理演示

举例演示：repo + prompt

```text
目标：开发一个简单的 Web 服务
Prompt: 请构建一个包含用户认证与资源管理的全栈项目，前端用 React，后端用 FastAPI，数据库用 SQLite。
```

Agent 会自己建项目结构、分层、甚至写测试！

---

## ⚙️ VibeCode 调优细节

### 什么是 MCP？

MCP（Model Context Protocol）是一种协议，旨在为大型语言模型（LLMs）提供结构化的上下文管理。

### 三大关键组件：

#### 1. [Context7](https://upstash.com/blog/context7-llmtxt-cursor)

- **功能**：注入最新 API 示例与文档，避免调用错误版本。
- **场景**：用 Claude / Cursor 开发时保障上下文一致性。

#### 2. [Code Rules](https://www.prompthub.us/blog/top-cursor-rules-for-coding-agents)

- **功能**：制定 AI 的编码风格指令。
- **场景**：规范生成代码风格，方便团队协作。

#### 3. [Mem0](https://mem0.ai/)

- **功能**：赋予 AI 持久记忆力。
- **场景**：长周期项目中自动“记住”上下文和偏好。

---

## 📌 行内落地例子

在 `navy` 项目中做设备管理 & 资源管理：

- 用了行内 commit 技巧记录 agent 行为。
- 协作和复盘变得超级高效！

---

## 🚀 VibeCode 实践流程

完整流程如下：

1. 💡 想法生成（ChatGPT DeepResearch）  
2. 📄 设计/需求文档（ChatGPT）  
3. 💻 编码实现（Gemini 2.5 Pro）  
4. 🧐 代码审查（Claude 3.7 Sonnet）  
5. 🧹 代码优化（Gemini）  
6. ✅ 代码测试（Claude）  
7. 📚 使用手册（Gemini）

AI 全流程参与，你就是产品主理人 + 技术负责人 + 项目经理！

