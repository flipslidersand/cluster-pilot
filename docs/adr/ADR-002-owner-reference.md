# ADR-002: Job に OwnerReference を設定して DataPipeline と紐付ける

- **日付**: 2026-06-22
- **状態**: Accepted

## 決定

生成する Kubernetes Job に `OwnerReference` を設定し、`DataPipeline` を Owner とする。

## 理由

- `DataPipeline` を削除すると Kubernetes が自動的に関連 Job を GC する（クリーンアップ不要）
- `List` + `LabelSelector` で DataPipeline に紐づく Job を効率的に取得できる
- Reconcile Loop で「この Job は誰のものか」を明示できるためデバッグが容易

## トレードオフ

- OwnerReference は同一 Namespace 内でのみ有効（cross-namespace は不可）
