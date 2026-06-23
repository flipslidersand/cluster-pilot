# Tech Stack — ClusterPilot

## 言語・バージョン

- Go 1.22+

## 主要パッケージ

| パッケージ                            | 役割                        | 選定理由                        |
| ------------------------------------- | --------------------------- | ------------------------------- |
| `sigs.k8s.io/controller-runtime`      | Reconcile Loop 基盤         | Kubebuilder の標準ランタイム    |
| `sigs.k8s.io/kubebuilder`             | CRD scaffold・コード生成    | Operator 開発の事実上の標準     |
| `k8s.io/client-go`                    | Kubernetes API クライアント | controller-runtime の依存       |
| `k8s.io/apimachinery`                 | Kubernetes 型定義           | 同上                            |
| `go.opentelemetry.io/otel`            | トレース・メトリクス        | Phase 6                         |
| `github.com/prometheus/client_golang` | Prometheus メトリクス       | Phase 6                         |
| `go.uber.org/zap`                     | 構造化ログ                  | controller-runtime と親和性高い |

## アーキテクチャ

```
kubectl apply -f pipeline.yaml
  ↓
[API Server] DataPipeline CRD
  ↓ (Watch)
[DataPipelineReconciler]
  ├── desiredJob() → Kubernetes Job spec 生成
  ├── createOrUpdate(job)
  ├── observeJob() → Job の Status を読む
  └── updateStatus() → DataPipeline.Status に反映
```

## 開発環境

```bash
# ローカル Kubernetes
kind create cluster --name clusterpilot

# CRD 生成
make generate && make manifests

# 実行 (クラスタ外)
make run
```
