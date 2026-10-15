---
id: expense-claims
title: { en: Expense claims, vi: Đề nghị thanh toán, ja: 経費申請 }
area: claims
roles: [employee, approver]
summary: { en: "Employees create claims and follow them until an approver approves or returns them." }
screens: [claim-list, claim-form, claim-detail]
flows: [submit-claim]
modules: [web-claims, app-data]
terms: [claim, receipt, return]
specs:
  - { label: "Demo app source", url: "https://github.com/lamngockhuong/muninn/tree/main/examples/demo/app" }
sources: [app/claims.html, app/claim-new.html, app/claim.html, app/app.js]
review: { status: draft }
---
## What it solves

Employees no longer send receipts by email. Each expense is one claim with an amount, a date and a
category, and its status is visible to the employee at all times.

## Business rules

- A new claim is **Submitted** immediately; there is no saved draft in the demo.
- Amounts are whole yen without separators. `12,000` is rejected; `12000` is accepted.
- An employee sees only their own claims.
- A returned claim shows the approver's note on its detail page.

## Status

```mermaid
stateDiagram-v2
  [*] --> Submitted: employee submits
  Submitted --> Approved: approver approves
  Submitted --> Returned: approver returns with a note
```
