---
id: app-data
title: { en: Data and session }
kind: package
paths: [app/app.js]
tables: [acme.claims (localStorage), acme.decisions (localStorage), acme.user (localStorage)]
sources: [app/app.js]
review: { status: draft }
---
## Responsibility

Seed claims, the signed-in user and role, and approver decisions. Every page loads this file; there is
no backend, so all state lives in the browser's localStorage.
