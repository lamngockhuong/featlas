---
id: approvals
title: { en: Approvals, vi: Phê duyệt, ja: 承認 }
area: approvals
roles: [approver]
summary: { en: "Approvers work through submitted claims, oldest first, and approve or return each one." }
screens: [approval-queue, claim-detail]
modules: [web-approvals, app-data]
terms: [claim, return]
sources: [app/approvals.html, app/claim.html, app/app.js]
review: { status: draft }
---
## What it solves

One queue for every submitted claim across all employees, so nothing waits in someone's inbox.

## Business rules

- The queue lists only **Submitted** claims, oldest expense date first.
- Approve and Return appear only for an approver, and only on a submitted claim.
- Returning a claim requires a note; cancelling the note prompt leaves the claim submitted.
