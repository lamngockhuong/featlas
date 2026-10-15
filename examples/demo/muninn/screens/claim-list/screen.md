---
id: claim-list
ref: EX-01
title: { en: My claims, vi: Đề nghị của tôi, ja: 経費申請一覧 }
feature: expense-claims
route: /claims.html
roles: [employee]
variants:
  - id: default
    label: { en: With claims, vi: Có đề nghị, ja: 申請あり }
    shot: shots/default.webp
    hotspots:
      - n: 1
        label: { en: Status filter, vi: Bộ lọc trạng thái, ja: 状態フィルター }
        desc: { en: "The filter value is kept in the URL (?status=), so a filtered list can be shared." }
        box: [81, 126, 115, 34]
      - n: 2
        label: { en: New claim, vi: Tạo đề nghị, ja: 新規申請 }
        box: [969, 124, 107, 38]
        kind: link
        to: claim-form
      - n: 3
        label: { en: Table row, vi: Dòng trong bảng, ja: 表の行 }
        desc: { en: "Click a row to open the claim; there is no separate open button." }
        box: [24, 227, 1052, 53]
        kind: hidden
        to: claim-detail
  - id: empty
    label: { en: No matching claims, vi: Không có đề nghị phù hợp, ja: 該当なし }
    shot: shots/empty.webp
    hotspots:
      - n: 1
        label: { en: Empty state }
        desc: { en: "Shown when the filter matches nothing. Choose All to see every claim again." }
        box: [24, 178, 1052, 118]
fields:
  - { label: Title, meaning: { en: Short description typed by the employee }, source: "claim.title" }
  - { label: Date, meaning: { en: Day the money was spent }, source: "claim.date" }
  - { label: Amount, meaning: { en: "Whole yen, shown with separators" }, source: "claim.amount" }
  - { label: Status, meaning: { en: "Submitted, Approved or Returned" }, source: "claim.status" }
tables: [acme.claims (localStorage)]
modules: [web-claims, app-data]
links:
  - { to: claim-detail, how: { en: Click a row } }
  - { to: claim-form, how: { en: New claim button } }
sources: [app/claims.html, app/app.js]
captured: { at: 2026-10-14, branch: main, backend: chromedp, viewport: 1366x768 }
review: { status: draft }
---
## Purpose

The employee's home page: every claim they created, newest seed data first, with its current status.

## Notes

- Only the signed-in employee's claims are listed; approvers are redirected to sign in again.
- A filter with no match shows the empty state rather than an empty table.
