---
id: sign-in
title: { en: Sign in, vi: Đăng nhập, ja: ログイン }
area: account
roles: [employee, approver]
summary: { en: "Everyone signs in with an email address; the address decides the role." }
screens: [sign-in]
modules: [app-data]
sources: [app/login.html, app/app.js]
review: { status: draft }
---
## What it solves

Both roles use the same sign-in page. There is no separate approver portal.

## Business rules

- Only `@acme.test` addresses can sign in. The password is shared and fixed in the demo.
- An address starting with `approver` gets the approver role; every other address is an employee.
- After signing in, employees land on **My claims** and approvers land on **Waiting for approval**.
