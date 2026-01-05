# Primind Central Backend

## モジュール

### Auth Module

- OpenID Connect (OIDC)認証
- セッション管理
- ユーザー管理

proto: `proto/auth/v1/auth.proto`

対応OIDC Provider:
- Google OIDC Provider

### Device Module

- デバイス登録・管理
- FCMトークン管理

proto: `proto/device/v1/device.proto`

連携: Auth Module（セッション検証）

### Task Module

- タスクCRUD
- リマインド登録・キャンセル

proto:
- `proto/task/v1/task.proto`
- `proto/remind/v1/remind.proto`
  time-mgmt連携用
- `proto/taskqueue/v1/taskqueue.proto`
  primind-tasks連携用

連携:
  - Auth Module（セッション検証）
  - Device Module（デバイス情報取得）
  - Remind Time Managemnt （リマインド登録・キャンセル）
  - Cloud Tasks / Primind Tasks

## 依存

- PostgreSQL v18
- Redis v8
- Remind Time Managemnt
- Cloud Tasks / Primind Tasks
