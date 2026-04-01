'use strict';

const API_ORDERS   = '/api/orders';
const API_PAYMENTS = '/api/payments';

let logCount = 0;

function addLog(method, endpoint, status, body) {
  logCount++;
  document.getElementById('log-count').textContent = `${logCount} entr${logCount === 1 ? 'y' : 'ies'}`;

  const empty = document.getElementById('log-empty');
  if (empty) empty.remove();

  const scroll = document.getElementById('log-scroll');
  const ts = new Date().toLocaleTimeString('en-US', { hour12: false });

  const statusClass = typeof status === 'number'
    ? (status >= 500 ? 's-5xx' : status >= 400 ? 's-4xx' : 's-2xx')
    : 's-err';

  const methodClass = `method-${method.toLowerCase()}`;
  const bodyStr = typeof body === 'string' ? body : JSON.stringify(body, null, 2);
  const entryId = `log-${logCount}`;

  const el = document.createElement('div');
  el.className = 'log-entry';
  el.innerHTML = `
    <div class="log-entry-head">
      <span class="log-ts">${ts}</span>
      <span class="log-method ${methodClass}">${method}</span>
      <span class="log-endpoint">${endpoint}</span>
      <span class="log-status ${statusClass}">${status}</span>
      <span class="log-toggle" onclick="toggleLogBody('${entryId}')">▾ body</span>
    </div>
    <div class="log-body" id="${entryId}" style="display:none">${escHtml(bodyStr)}</div>
  `;
  scroll.appendChild(el);
  scroll.scrollTop = scroll.scrollHeight;
}

function toggleLogBody(id) {
  const el = document.getElementById(id);
  if (el) el.style.display = el.style.display === 'none' ? 'block' : 'none';
}

function escHtml(s) {
  return s.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;');
}

async function apiFetch(method, url, body, headers = {}) {
  const opts = {
    method,
    headers: { 'Content-Type': 'application/json', ...headers },
  };
  if (body !== undefined) opts.body = JSON.stringify(body);

  let status = 'ERR';
  let data;

  try {
    const res = await fetch(url, opts);
    status = res.status;
    try { data = await res.json(); } catch { data = { error: 'empty or non-JSON response' }; }
    addLog(method, url, status, data);
    return { ok: res.ok, status, data };
  } catch (err) {
    data = { error: err.message || 'Network error — service may be unavailable' };
    addLog(method, url, 'ERR', data);
    return { ok: false, status: 0, data };
  }
}

function setLoading(btnId, spinId, loading) {
  const btn  = document.getElementById(btnId);
  const spin = document.getElementById(spinId);
  const text = btn.querySelector('.btn-text');
  btn.disabled = loading;
  spin.classList.toggle('hidden', !loading);
  text.style.opacity = loading ? '.5' : '1';
}

function statusColorClass(status) {
  if (!status) return '';
  const s = status.toLowerCase();
  if (s === 'paid' || s === 'authorized') return 'status-paid';
  if (s === 'failed' || s === 'declined') return 'status-failed';
  if (s === 'pending')                    return 'status-pending';
  if (s === 'cancelled')                  return 'status-cancelled';
  return '';
}

function chipClass(status) {
  if (!status) return 'chip-blue';
  const s = status.toLowerCase();
  if (s === 'paid' || s === 'authorized') return 'chip-green';
  if (s === 'failed' || s === 'declined') return 'chip-red';
  if (s === 'pending')                    return 'chip-yellow';
  if (s === 'cancelled')                  return 'chip-gray';
  return 'chip-blue';
}

function httpChipClass(httpStatus) {
  if (httpStatus >= 500) return 'chip-red';
  if (httpStatus >= 400) return 'chip-yellow';
  if (httpStatus >= 200) return 'chip-green';
  return 'chip-red';
}

function renderResult(boxId, httpStatus, data) {
  const box = document.getElementById(boxId);
  box.className = 'result-box';

  const domainStatus = data?.status || data?.order?.status;
  const colorClass   = httpStatus === 0 || httpStatus >= 500
    ? (httpStatus === 503 ? 'status-503' : 'status-error')
    : statusColorClass(domainStatus);

  if (colorClass) box.classList.add(colorClass);
  box.classList.remove('hidden');

  const chipCls = httpStatus === 0 ? 'chip-red' : httpChipClass(httpStatus);
  const label   = httpStatus === 0 ? 'Network Error' : `HTTP ${httpStatus}`;

  box.innerHTML = `
    <div class="result-status-bar">
      <span class="log-ts">${new Date().toLocaleTimeString('en-US',{hour12:false})}</span>
      <span class="status-chip ${chipCls}">${label}</span>
      ${domainStatus ? `<span class="status-chip ${chipClass(domainStatus)}">${domainStatus}</span>` : ''}
    </div>
    <pre class="result-json">${escHtml(JSON.stringify(data, null, 2))}</pre>
  `;
}

document.getElementById('form-create').addEventListener('submit', async (e) => {
  e.preventDefault();
  setLoading('btn-create', 'spin-create', true);

  const customerID = document.getElementById('customerID').value.trim();
  const itemName   = document.getElementById('itemName').value.trim();
  const amount     = parseInt(document.getElementById('amount').value, 10);
  const idempKey   = document.getElementById('idempKey').value.trim();

  const headers = {};
  if (idempKey) headers['Idempotency-Key'] = idempKey;

  const { status, data } = await apiFetch('POST', API_ORDERS,
    { customer_id: customerID, item_name: itemName, amount },
    headers
  );

  setLoading('btn-create', 'spin-create', false);
  renderResult('create-result', status, data);

  if (data?.id) {
    ['getOrderID','cancelOrderID','paymentOrderID'].forEach(id => {
      document.getElementById(id).value = data.id;
    });
  }
});

document.getElementById('btn-get-order').addEventListener('click', async () => {
  const id = document.getElementById('getOrderID').value.trim();
  if (!id) return;
  setLoading('btn-get-order', 'spin-get', true);
  const { status, data } = await apiFetch('GET', `${API_ORDERS}/${id}`);
  setLoading('btn-get-order', 'spin-get', false);
  renderResult('get-order-result', status, data);
});

document.getElementById('btn-cancel-order').addEventListener('click', async () => {
  const id = document.getElementById('cancelOrderID').value.trim();
  if (!id) return;
  setLoading('btn-cancel-order', 'spin-cancel', true);
  const { status, data } = await apiFetch('PATCH', `${API_ORDERS}/${id}/cancel`);
  setLoading('btn-cancel-order', 'spin-cancel', false);
  renderResult('cancel-result', status, data);
});

document.getElementById('btn-get-payment').addEventListener('click', async () => {
  const id = document.getElementById('paymentOrderID').value.trim();
  if (!id) return;
  setLoading('btn-get-payment', 'spin-payment', true);
  const { status, data } = await apiFetch('GET', `${API_PAYMENTS}/${id}`);
  setLoading('btn-get-payment', 'spin-payment', false);
  renderResult('payment-result', status, data);
});

document.getElementById('btn-clear-log').addEventListener('click', () => {
  const scroll = document.getElementById('log-scroll');
  scroll.innerHTML = '<div class="log-empty" id="log-empty">No activity yet — run a request above to see logs here.</div>';
  logCount = 0;
  document.getElementById('log-count').textContent = '0 entries';
});

const SCENARIOS = {
  normal: {
    customerID: 'cust-demo-1',
    itemName:   'Laptop Pro',
    amount:     15000,
    idempKey:   '',
  },
  overlimit: {
    customerID: 'cust-demo-2',
    itemName:   'Luxury Yacht',
    amount:     150000,
    idempKey:   '',
  },
  tiny: {
    customerID: 'cust-demo-3',
    itemName:   'Sticker Pack',
    amount:     100,
    idempKey:   '',
  },
  zero: {
    customerID: 'cust-demo-4',
    itemName:   'Free Item',
    amount:     0,
    idempKey:   '',
  },
  idempotency: {
    customerID: 'cust-idem-1',
    itemName:   'Idempotent Book',
    amount:     2500,
    idempKey:   'idem-key-fixed-demo-001',
  },
};

document.querySelectorAll('.scenario-btn').forEach(btn => {
  btn.addEventListener('click', () => {
    const key = btn.dataset.scenario;
    const s   = SCENARIOS[key];
    if (!s) return;
    document.getElementById('customerID').value = s.customerID;
    document.getElementById('itemName').value   = s.itemName;
    document.getElementById('amount').value     = s.amount;
    document.getElementById('idempKey').value   = s.idempKey;
    document.getElementById('card-create').scrollIntoView({ behavior: 'smooth', block: 'nearest' });
  });
});

document.getElementById('getOrderID').addEventListener('keydown', e => {
  if (e.key === 'Enter') document.getElementById('btn-get-order').click();
});
document.getElementById('cancelOrderID').addEventListener('keydown', e => {
  if (e.key === 'Enter') document.getElementById('btn-cancel-order').click();
});
document.getElementById('paymentOrderID').addEventListener('keydown', e => {
  if (e.key === 'Enter') document.getElementById('btn-get-payment').click();
});
