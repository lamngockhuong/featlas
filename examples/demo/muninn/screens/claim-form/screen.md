---
id: claim-form
title: { en: New claim, vi: Tạo đề nghị, ja: 新規申請 }
feature: expense-claims
route: /claim-new.html
roles: [employee]
variants:
  - id: default
    label: { en: Empty form }
    shot: shots/default.webp
    hotspots:
      - n: 1
        label: { en: Title }
        desc: { en: "Up to 80 characters." }
        box: [233, 149, 360, 36]
      - n: 2
        label: { en: Amount }
        desc: { en: "Whole yen without separators. 12,000 is rejected by the browser." }
        box: [233, 201, 360, 36]
      - n: 3
        label: { en: Date }
        box: [233, 253, 360, 38]
      - n: 4
        label: { en: Category }
        desc: { en: "Travel, Meals, Training or Supplies." }
        box: [233, 307, 360, 34]
      - n: 5
        label: { en: Receipt }
        desc: { en: "Optional. The demo does not store the file." }
        box: [233, 357, 360, 38]
      - n: 6
        label: { en: Submit for approval }
        desc: { en: "Creates the claim as Submitted and opens it. There is no draft." }
        box: [49, 419, 173, 38]
        to: claim-detail
fields:
  - { label: Title, meaning: { en: Short description }, source: "claim.title" }
  - { label: "Amount (¥)", meaning: { en: Amount in whole yen }, source: "claim.amount" }
  - { label: Date, meaning: { en: Day of the expense }, source: "claim.date" }
  - { label: Category, meaning: { en: Expense category }, source: "claim.category" }
  - { label: Receipt, meaning: { en: Proof of payment (not stored) } }
tables: [acme.claims (localStorage)]
modules: [web-claims, app-data]
links:
  - { to: claim-detail, how: { en: Submit for approval } }
  - { to: claim-list, how: { en: Cancel } }
sources: [app/claim-new.html, app/app.js]
captured: { at: 2026-10-14, branch: main, backend: chromedp, viewport: 1366x768 }
review: { status: draft }
---
## Purpose

Create one claim for one expense.

## Notes

All fields except Receipt are required; the browser blocks submission until they are filled.
