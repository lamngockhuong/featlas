---
id: claim-detail
title: { en: Claim, vi: Chi tiết đề nghị, ja: 申請詳細 }
feature: expense-claims
route: /claim.html?id=<claim id>
roles: [employee, approver]
variants:
  - id: returned
    label: { en: Returned claim (employee) }
    shot: shots/returned.webp
    hotspots:
      - n: 1
        label: { en: Status }
        box: [233, 149, 71, 24]
      - n: 2
        label: { en: Approver note }
        desc: { en: "Only present on returned claims: what the employee must fix." }
        box: [233, 313, 818, 20]
  - id: approver
    label: { en: Submitted claim (approver) }
    shot: shots/approver.webp
    hotspots:
      - n: 1
        label: { en: Approve }
        desc: { en: "Marks the claim Approved immediately; there is no confirmation." }
        box: [49, 325, 92, 38]
      - n: 2
        label: { en: Return to employee }
        desc: { en: "Asks for a note. Cancelling the prompt keeps the claim Submitted." }
        box: [153, 325, 173, 38]
fields:
  - { label: Status, meaning: { en: Current claim status }, source: "claim.status" }
  - { label: Amount, source: "claim.amount" }
  - { label: Date, source: "claim.date" }
  - { label: Category, source: "claim.category" }
  - { label: Employee, meaning: { en: Address of the claim owner }, source: "claim.owner" }
  - { label: Approver note, meaning: { en: Reason for returning }, source: "decision.note" }
tables: [acme.claims (localStorage), acme.decisions (localStorage)]
modules: [web-claims, web-approvals, app-data]
sources: [app/claim.html, app/app.js]
captured: { at: 2026-10-14, branch: main, backend: chromedp, viewport: 1366x768 }
review: { status: draft }
---
## Purpose

One claim with everything an approver needs to decide. Employees and approvers open the same page;
only approvers see the actions, and only while the claim is Submitted.
