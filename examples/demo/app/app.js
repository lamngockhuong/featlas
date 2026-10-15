// Acme Expense: a fake expense-claim app used as the muninn demo and capture target.
// All data lives in this file and in localStorage; there is no backend.

const SEED_CLAIMS = [
  { id: 'c-1001', title: 'Taxi to client office', amount: 4200, date: '2026-09-28', category: 'travel', status: 'submitted', owner: 'emma@acme.test' },
  { id: 'c-1002', title: 'Team lunch', amount: 18500, date: '2026-09-30', category: 'meals', status: 'approved', owner: 'emma@acme.test' },
  { id: 'c-1003', title: 'Conference ticket', amount: 52000, date: '2026-10-02', category: 'training', status: 'returned', owner: 'emma@acme.test', note: 'Attach the invoice, not the order confirmation.' },
  { id: 'c-1004', title: 'Train to Osaka', amount: 14450, date: '2026-10-05', category: 'travel', status: 'submitted', owner: 'liam@acme.test' },
];

const CATEGORIES = { travel: 'Travel', meals: 'Meals', training: 'Training', supplies: 'Supplies' };
const STATUS_LABELS = { draft: 'Draft', submitted: 'Submitted', approved: 'Approved', returned: 'Returned' };

function currentUser() {
  const email = localStorage.getItem('acme.user');
  if (!email) return null;
  return { email, role: email.startsWith('approver') ? 'approver' : 'employee' };
}

function requireUser(role) {
  const user = currentUser();
  if (!user || (role && user.role !== role)) {
    location.href = 'login.html';
    throw new Error('not signed in');
  }
  return user;
}

function allClaims() {
  const added = JSON.parse(localStorage.getItem('acme.claims') || '[]');
  const decisions = JSON.parse(localStorage.getItem('acme.decisions') || '{}');
  return [...SEED_CLAIMS, ...added].map((c) => (decisions[c.id] ? { ...c, ...decisions[c.id] } : c));
}

function saveClaim(claim) {
  const added = JSON.parse(localStorage.getItem('acme.claims') || '[]');
  added.push(claim);
  localStorage.setItem('acme.claims', JSON.stringify(added));
}

function decide(id, status, note) {
  const decisions = JSON.parse(localStorage.getItem('acme.decisions') || '{}');
  decisions[id] = { status, note };
  localStorage.setItem('acme.decisions', JSON.stringify(decisions));
}

function yen(n) { return '¥' + n.toLocaleString('en-US'); }

function badge(status) { return `<span class="badge ${status}">${STATUS_LABELS[status]}</span>`; }

function renderHeader(active) {
  const user = currentUser();
  const links = user && user.role === 'approver'
    ? [['approvals.html', 'Approvals', 'approvals']]
    : [['claims.html', 'My claims', 'claims']];
  document.querySelector('header').innerHTML = `
    <span class="brand">Acme Expense</span>
    <nav>${links.map(([href, label, key]) => `<a href="${href}" class="${key === active ? 'active' : ''}">${label}</a>`).join('')}</nav>
    <span class="user">${user ? user.email : ''} · <a href="login.html" id="sign-out" style="color:inherit">Sign out</a></span>`;
  document.getElementById('sign-out').addEventListener('click', () => localStorage.removeItem('acme.user'));
}
