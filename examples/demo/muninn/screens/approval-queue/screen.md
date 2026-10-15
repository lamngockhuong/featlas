---
id: approval-queue
title: { en: Waiting for approval, vi: Chờ phê duyệt, ja: 承認待ち }
feature: approvals
route: /approvals.html
roles: [approver]
variants:
  - id: default
    label: { en: Claims waiting }
    shot: shots/default.webp
    hotspots:
      - n: 1
        label: { en: Queue }
        desc: { en: "Every Submitted claim from every employee, oldest expense date first." }
        box: [24, 124, 1052, 196]
      - n: 2
        label: { en: Table row }
        desc: { en: "Click a row to open the claim and decide." }
        box: [24, 173, 1052, 49]
        kind: hidden
        to: claim-detail
fields:
  - { label: Employee, source: "claim.owner" }
  - { label: Title, source: "claim.title" }
  - { label: Date, source: "claim.date" }
  - { label: Amount, source: "claim.amount" }
tables: [acme.claims (localStorage)]
modules: [web-approvals, app-data]
links:
  - { to: claim-detail, how: { en: Click a row } }
sources: [app/approvals.html, app/app.js]
captured: { at: 2026-10-14, branch: main, backend: chromedp, viewport: 1366x768 }
review: { status: draft }
---
## Purpose

The approver's home page. When it is empty, it says "Nothing to approve."
