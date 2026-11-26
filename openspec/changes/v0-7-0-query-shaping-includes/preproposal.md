---
schema: gentle-ai.sdd-preproposal/v1
change: v0-7-0-query-shaping-includes
revision: 1
research_outcome: done
proposal_ready: false
---

# Pre-Proposal State: v0.7.0 Query Shaping Includes

## Exploration

- Outcome: ready for proposal
- Reference: `openspec/changes/v0-7-0-query-shaping-includes/exploration.md`
- Finding: use explicit query-shape profiles; keep default list/get endpoints shallow unless the author opts into deeper includes; query shaping should be read-side only and config-driven.

## Research Request

- Classes requested: documentation
- Classes admitted: documentation
- Classes not requested: open-web
- Evidence reference: `openspec/changes/v0-7-0-query-shaping-includes/research.md`
- Outcome: done

## Product Decisions

- Status: pending
- Pending confirmation: exact safety policy for multiple sibling collection includes: reject, require explicit split-query mode, or allow with generated warning.
- Pending confirmation: whether the WebAPI shape selector should be exposed for generated endpoints in v0.7.0 or remain config-only initially.

## Proposal Readiness

- Research evidence valid: yes
- Research outcome done: yes
- Product decisions confirmed: no
- OpenSpec evidence readback required: yes
- Proposal ready: no

## Handoff Notes

- Favor explicit named query-shape profiles.
- Keep EF Core include/split-query/projection mechanics inside infrastructure.
- Generate application handlers and WebAPI surfaces against stable shape names, not raw include paths.
- Generate acyclic, shape-specific DTOs for read models; do not mirror the full entity graph by default.
