# Spec — ClusterPilot

## プロジェクトの目的

独自の `DataPipeline` CRD を定義し、Kubernetes Job の生成・再実行・状態更新・クリーンアップを自動化する Kubernetes Operator。
Kubebuilder / controller-runtime を使った Reconcile Loop の実装を習得する。

## 利用イメージ

```yaml
apiVersion: clusterpilot.dev/v1
kind: DataPipeline
metadata:
  name: vehicle-import
spec:
  schedule: "0 * * * *"
  image: vehicle-importer:latest
  retries: 3
  timeout: 10m
```

```bash
kubectl apply -f pipeline.yaml
kubectl get datapipeline vehicle-import
# NAME             STATUS    RUNS  LAST-RUN
# vehicle-import   Running   12    2026-06-22T10:00:00Z
```

## MVP の境界線

### やること (Phase 1〜4)

- `DataPipeline` CRD の定義 (kubebuilder)
- Reconcile Loop: CRD → Kubernetes Job 生成
- Job 成功/失敗を `DataPipeline.Status` に反映
- 失敗時の再実行 (`retries` 上限まで)
- タイムアウト検出と Job 削除

### やらないこと (Phase 1)

- Cron スケジュール実行
- ローリング更新 / カナリア
- Webhook Validation
- Prometheus メトリクス

## 成功条件

| Phase   | 完成条件                                                |
| ------- | ------------------------------------------------------- |
| Phase 1 | kubebuilder scaffold + CRD を kind クラスタに適用できる |
| Phase 2 | Reconcile Loop が DataPipeline → Job を生成する         |
| Phase 3 | Job 完了/失敗を DataPipeline.Status に反映する          |
| Phase 4 | retries 上限まで自動再実行、timeout で Job を削除       |
| Phase 5 | `schedule` フィールドで Cron 実行                       |
| Phase 6 | Prometheus メトリクス + OTel トレース                   |
