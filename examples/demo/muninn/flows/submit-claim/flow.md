---
id: submit-claim
title: { en: Submit an expense claim, vi: Gửi đề nghị thanh toán, ja: 経費を申請する }
feature: expense-claims
role: employee
goal: { en: "Create a claim for a client dinner and send it for approval." }
start: claim-list
end: claim-detail
steps:
  - n: 1
    action: click
    label: { en: New claim button }
    screen: claim-list
    variant: default
    box: [969, 124, 107, 38]
  - n: 2
    action: type
    label: { en: Claim details }
    shot: shots/step-2.webp
    box: [49, 149, 1002, 246]
    sample:
      - { label: Title, value: "Client dinner" }
      - { label: "Amount (¥)", value: "12000", note: { en: "whole yen, no separators" } }
      - { label: Date, value: "2026-10-12" }
      - { label: Category, value: "Meals" }
  - n: 3
    action: click
    label: { en: Submit for approval }
    shot: shots/step-2.webp
    box: [49, 419, 173, 38]
  - n: 4
    action: result
    label: { en: The claim opens with status Submitted }
    shot: shots/step-4.webp
    box: [233, 149, 78, 24]
sources: [app/claims.html, app/claim-new.html, app/claim.html]
review: { status: draft }
---
The claim now appears in the approver's queue. The employee is not notified when it is decided; they
see the new status on **My claims**.
