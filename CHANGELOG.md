# Changelog

## v0.1.0 — 2026-10-27

- 首个带版本标签的源码交付，包含现有 fault、stream、tool 及 Eino/Hertz 适配层。
- 经权利人确认，增加 MIT LICENSE 和许可边界说明；第三方依赖保留各自许可证。
- 相对 f91829f 仅增加许可及文档，不修改 Go API、运行代码或依赖版本。
- 发布前 `go test ./...`、`go vet ./...`、`git diff --check` 通过。

v0 阶段不承诺 API 永久稳定；此版本不是客服系统 1.0.0，也不包含预编译多平台包。
