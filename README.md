# cluster-pilot

Kubernetes Operator for automated DataPipeline cluster lifecycle management and self-healing (Go + Kubebuilder).

DataPipeline クラスターのライフサイクル管理と自己修復を自動化する Kubernetes Operator。

> 🟡 **Scaffold Phase** — 基本設計と MVP 実装準備中

## Status / ステータス

| Item | State | 状態 |
|------|------|------|
| Phase / フェーズ | Scaffold | Scaffold |
| MVP Start / 実装予定 | Q3 2026 | 2026-Q3 MVP開始 |
| Tests / テスト | Not yet | 未実装 |
| Docs / ドキュメント | Planned | 企画中 |

## Tech Stack / 技術スタック

- **Language / 言語:** Go
- **Framework:** Kubebuilder
- **Target / 対象:** Kubernetes 1.27+

## Roadmap / 実装ロードマップ

1. **Phase 1 (Q3 2026)** — CRD definition + basic reconciler / CRD 定義 + 基本 Reconciler
2. **Phase 2 (Q4 2026)** — Self-healing logic + status conditions / 自己修復ロジック + ステータス条件
3. **Phase 3 (Q1 2027)** — Scaling policies + health probes / スケーリングポリシー + ヘルスプローブ
4. **Phase 4 (Q2 2027)** — Integration tests + docs / 統合テスト + ドキュメント

## Notes / 注意事項

- **In Development / 開発中**: API/design may change without notice / API・設計は予告なく変更される可能性があります
- **No tests yet / テスト未実装**: Quality not guaranteed until Phase 2 / 本格実装まで品質保証していません

---

Track progress in [GitHub Issues](https://github.com/flipslidersand/cluster-pilot/issues).
