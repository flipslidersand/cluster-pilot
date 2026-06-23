# ADR-001: Operator 基盤に Kubebuilder + controller-runtime を使う

- **日付**: 2026-06-22
- **状態**: Accepted

## 決定

`kubebuilder` でプロジェクトを scaffold し `controller-runtime` で Reconcile Loop を実装する。

## 理由

- CRD・RBAC・Webhook のボイラープレートを自動生成でき、本質的な Reconcile ロジックに集中できる
- `controller-runtime` の `Manager` が Leader Election・Health Check・メトリクスを提供する
- Kubernetes Operator の学習リソースが Kubebuilder ベースで最も充実している

## トレードオフ

- 生成されるコードが多く、全体像の把握に時間がかかる
- `make generate` を忘れると型定義とコードがずれる
