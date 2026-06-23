# Implementation Guide — ClusterPilot

## Phase 1: kubebuilder Scaffold + CRD（1週）

```bash
kubebuilder init --domain clusterpilot.dev --repo github.com/flipslidersand/cluster-pilot
kubebuilder create api --group clusterpilot --version v1 --kind DataPipeline
make generate && make manifests
kind create cluster --name clusterpilot
make install   # CRD をクラスタに適用
```

**完成条件**: `kubectl get crd datapipelines.clusterpilot.dev` が通る

---

## Phase 2: Reconcile Loop → Job 生成（1週）

- `internal/controller/datapipeline_controller.go` に `Reconcile()` を実装
- `desiredJob()` で Kubernetes Job spec を生成
- `createOrUpdate()` で Job を apply
- `OwnerReference` を設定して DataPipeline 削除時に Job も消える

**完成条件**: `kubectl apply -f pipeline.yaml` で Job が生成される

---

## Phase 3: Status 反映（1週）

- Job の `.Status.Conditions` を Watch
- 完了/失敗を `DataPipeline.Status.Phase` と `LastResult` に書き込む
- `kubectl get datapipeline` で STATUS が更新される

---

## Phase 4: Retry + Timeout（1週）

- 失敗時に `RunCount < Retries` なら新しい Job を生成
- `Spec.Timeout` を超えた Job を削除して `Failed` に遷移

---

## Phase 5: Cron スケジュール（1週）

- `Spec.Schedule` の cron 式をパースして次回実行時刻を計算
- `ctrl.Result{RequeueAfter: nextRun}` で再エンキュー

---

## Phase 6: Prometheus + OTel（3日）

- `controller_reconcile_total` / `pipeline_duration_seconds` を Prometheus で公開
- OTel Span で Reconcile の実行時間を計測
