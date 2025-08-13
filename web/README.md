# Web 前端（Vue 3 + Vite + TypeScript）

本目录为 MyFlowHub 的管理前端，使用 Naive UI、Pinia、Vue Router。

## 环境要求

- Node.js: 20.19+ 或 22.12+
- VS Code 推荐安装 Volar 扩展

## 安装与启动（PowerShell）

```powershell
cd d:\rj\MyFlowHub\web
npm install
npm run dev
```

生产构建：

```powershell
npm run build
```

## 运行说明

- 管理端通过 manager 提供的 REST API 工作（默认 http://localhost:8090/api）。
- 登录：访问 /login，成功后前端会保存 token 并自动为请求附加 Authorization 头。
- 权限：
  - 未登录用户仅可访问 /login。
  - 仅拥有 admin.manage 的用户可访问“用户管理”等管理员页面。

## 目录结构

- src/views/manage/UsersView.vue 用户管理（含权限编辑弹窗）
- src/stores/auth.ts 登录状态与权限快照
- src/services/api.ts API 封装，自动附带 token

## 常见问题

- Windows 下安装失败（EPERM unlink esbuild.exe）：关闭杀软/重启终端后重试；必要时删除 node_modules 与 package-lock.json 重新安装。
- 报错 run-p 未找到：项目使用 npm-run-all2，确保 npm install 成功；也可直接 npm run build-only 先构建。
