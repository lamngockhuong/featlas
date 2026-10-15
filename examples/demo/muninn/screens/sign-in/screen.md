---
id: sign-in
title: { en: Sign in, vi: Đăng nhập, ja: ログイン }
feature: sign-in
route: /login.html
roles: [employee, approver]
variants:
  - id: default
    label: { en: Empty form }
    shot: shots/default.webp
    hotspots:
      - n: 1
        label: { en: Email }
        desc: { en: "Must end in @acme.test. An address starting with approver signs in as an approver." }
        box: [528, 197, 310, 36]
      - n: 2
        label: { en: Password }
        desc: { en: "The same for every demo account." }
        box: [528, 277, 310, 36]
      - n: 3
        label: { en: Sign in }
        desc: { en: "Wrong credentials show an error under the password; the fields are kept." }
        box: [528, 325, 310, 38]
        to: claim-list
fields:
  - { label: Email, meaning: { en: "Account address; decides the role" } }
  - { label: Password, meaning: { en: Shared demo password } }
modules: [app-data]
links:
  - { to: claim-list, how: { en: Sign in as an employee } }
  - { to: approval-queue, how: { en: Sign in as an approver } }
sources: [app/login.html, app/app.js]
captured: { at: 2026-10-14, branch: main, backend: chromedp, viewport: 1366x768 }
review: { status: draft }
---
## Purpose

The only entry point of the app. The signed-in address is kept in the browser until **Sign out**.
