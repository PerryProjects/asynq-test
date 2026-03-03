package web

// indexHTML is the htmx-powered single-page dashboard template.
const indexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Asynq Multi-Pod Prototype</title>
    <script src="https://unpkg.com/htmx.org@1.9.12"></script>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: #f3f4f6; color: #111827; padding: 1rem; }
        h1 { font-size: 1.5rem; margin-bottom: 0.5rem; }
        h2 { font-size: 1.15rem; margin: 1.2rem 0 0.5rem; color: #374151; }
        a { color: #2563eb; }
        .container { max-width: 1100px; margin: 0 auto; }
        .header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem; }
        .badge { background: #10b981; color: #fff; padding: 2px 8px; border-radius: 9999px; font-size: 0.75rem; }
        .badge.paused { background: #f59e0b; }

        /* Cards */
        .card { background: #fff; border-radius: 8px; box-shadow: 0 1px 3px rgba(0,0,0,0.1); padding: 1rem; margin-bottom: 1rem; }
        .grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 1rem; }

        /* Tables */
        table { width: 100%; border-collapse: collapse; font-size: 0.875rem; }
        th, td { padding: 0.5rem 0.75rem; text-align: left; border-bottom: 1px solid #e5e7eb; }
        th { background: #f9fafb; font-weight: 600; }
        tr:hover { background: #f3f4f6; }

        /* Forms */
        .form-group { margin-bottom: 0.75rem; }
        label { display: block; font-weight: 500; margin-bottom: 0.25rem; font-size: 0.875rem; }
        input, select, textarea { width: 100%; padding: 0.4rem 0.6rem; border: 1px solid #d1d5db; border-radius: 6px; font-size: 0.875rem; }
        textarea { resize: vertical; min-height: 80px; font-family: monospace; }
        .btn { padding: 0.5rem 1rem; border: none; border-radius: 6px; cursor: pointer; font-size: 0.875rem; font-weight: 500; }
        .btn-primary { background: #2563eb; color: #fff; }
        .btn-primary:hover { background: #1d4ed8; }
        .btn-sm { padding: 0.25rem 0.6rem; font-size: 0.75rem; }
        .btn-yellow { background: #f59e0b; color: #fff; }
        .btn-green { background: #10b981; color: #fff; }
        .btn-red { background: #ef4444; color: #fff; }

        /* Result area */
        #enqueue-result { margin-top: 0.75rem; padding: 0.5rem; border-radius: 6px; font-size: 0.85rem; }
        #enqueue-result:not(:empty) { background: #ecfdf5; border: 1px solid #6ee7b7; }
        .error { background: #fef2f2 !important; border-color: #fca5a5 !important; color: #991b1b; }

        /* Auto-refresh indicator */
        .refresh { font-size: 0.75rem; color: #9ca3af; }
    </style>
</head>
<body>
<div class="container">
    <div class="header">
        <div>
            <h1>Asynq Multi-Pod Prototype</h1>
            <span class="refresh">Dashboard auto-refreshes every 5s &bull; <a href="/monitoring" target="_blank">Open Asynqmon &rarr;</a></span>
        </div>
    </div>

    <!-- ── Live Dashboard (auto-refresh) ─────────────────────────── -->
    <div hx-get="/" hx-trigger="every 5s" hx-select="#live-data" hx-target="#live-data" hx-swap="outerHTML">
        <div id="live-data">
            <!-- Queues -->
            <h2>Queues</h2>
            <div class="card">
                <table>
                    <thead>
                        <tr>
                            <th>Queue</th>
                            <th>Size</th>
                            <th>Pending</th>
                            <th>Active</th>
                            <th>Scheduled</th>
                            <th>Retry</th>
                            <th>Archived</th>
                            <th>Completed</th>
                            <th>Status</th>
                            <th>Action</th>
                        </tr>
                    </thead>
                    <tbody>
                        {{range .Queues}}
                        <tr>
                            <td><strong>{{.Name}}</strong></td>
                            <td>{{.Size}}</td>
                            <td>{{.Pending}}</td>
                            <td>{{.Active}}</td>
                            <td>{{.Scheduled}}</td>
                            <td>{{.Retry}}</td>
                            <td>{{.Archived}}</td>
                            <td>{{.Completed}}</td>
                            <td>
                                {{if .Paused}}
                                    <span class="badge paused">Paused</span>
                                {{else}}
                                    <span class="badge">Active</span>
                                {{end}}
                            </td>
                            <td>
                                {{if .Paused}}
                                    <button class="btn btn-sm btn-green"
                                        hx-post="/api/queues/{{.Name}}/unpause"
                                        hx-swap="none">Unpause</button>
                                {{else}}
                                    <button class="btn btn-sm btn-yellow"
                                        hx-post="/api/queues/{{.Name}}/pause"
                                        hx-swap="none">Pause</button>
                                {{end}}
                            </td>
                        </tr>
                        {{else}}
                        <tr><td colspan="10">No queues found — tasks have not been enqueued yet.</td></tr>
                        {{end}}
                    </tbody>
                </table>
            </div>

            <!-- Workers -->
            <h2>Connected Workers</h2>
            <div class="card">
                <table>
                    <thead>
                        <tr>
                            <th>ID</th>
                            <th>Host</th>
                            <th>Concurrency</th>
                            <th>Active Workers</th>
                            <th>Status</th>
                        </tr>
                    </thead>
                    <tbody>
                        {{range .Servers}}
                        <tr>
                            <td><code>{{.ID}}</code></td>
                            <td>{{.Host}}</td>
                            <td>{{.Concurrency}}</td>
                            <td>{{.ActiveCount}}</td>
                            <td><span class="badge">{{.Status}}</span></td>
                        </tr>
                        {{else}}
                        <tr><td colspan="5">No workers connected yet.</td></tr>
                        {{end}}
                    </tbody>
                </table>
            </div>
        </div>
    </div>

    <!-- ── Enqueue Task Form ─────────────────────────────────────── -->
    <h2>Trigger Task</h2>
    <div class="card">
        <form id="enqueue-form">
            <div class="grid">
                <div>
                    <div class="form-group">
                        <label for="task-type">Task Type</label>
                        <select id="task-type" name="type" onchange="updatePayload()">
                            <option value="email:deliver">email:deliver</option>
                            <option value="image:resize">image:resize</option>
                            <option value="report:generate">report:generate</option>
                            <option value="webhook:send">webhook:send</option>
                            <option value="notification:send">notification:send</option>
                        </select>
                    </div>
                    <div class="form-group">
                        <label for="task-queue">Queue (optional override)</label>
                        <select id="task-queue" name="queue">
                            <option value="">— use task default —</option>
                            <option value="critical">critical</option>
                            <option value="default">default</option>
                            <option value="low">low</option>
                        </select>
                    </div>
                    <div class="form-group">
                        <label for="task-delay">Delay (seconds)</label>
                        <input type="number" id="task-delay" name="delay_seconds" value="0" min="0">
                    </div>
                    <div class="form-group">
                        <label for="task-retry">Max Retry</label>
                        <input type="number" id="task-retry" name="max_retry" value="0" min="0">
                    </div>
                    <div class="form-group">
                        <label for="task-unique">Unique TTL (seconds)</label>
                        <input type="number" id="task-unique" name="unique_ttl_seconds" value="0" min="0">
                    </div>
                </div>
                <div>
                    <div class="form-group">
                        <label for="task-payload">Payload (JSON)</label>
                        <textarea id="task-payload" name="payload" rows="8"></textarea>
                    </div>
                    <button type="button" class="btn btn-primary" onclick="enqueueTask()">Enqueue Task</button>
                </div>
            </div>
        </form>
        <div id="enqueue-result"></div>
    </div>

    <!-- ── Quick Batch Triggers ──────────────────────────────────── -->
    <h2>Quick Actions</h2>
    <div class="card">
        <div style="display:flex;gap:0.5rem;flex-wrap:wrap;">
            <button class="btn btn-primary" onclick="quickEnqueue('email:deliver', {to:'user@test.com', subject:'Hello', body:'Test email body'})">
                Send Email
            </button>
            <button class="btn btn-primary" onclick="quickEnqueue('image:resize', {url:'https://example.com/photo.jpg', width:800, height:600})">
                Resize Image
            </button>
            <button class="btn btn-primary" onclick="quickEnqueue('report:generate', {report_type:'weekly', start_date:'2025-01-01', end_date:'2025-01-07'})">
                Generate Report
            </button>
            <button class="btn btn-primary" onclick="quickEnqueue('webhook:send', {url:'https://httpbin.org/post', method:'POST', simulate_code:200})">
                Webhook (200 OK)
            </button>
            <button class="btn btn-yellow" onclick="quickEnqueue('webhook:send', {url:'https://httpbin.org/post', method:'POST', simulate_code:500})">
                Webhook (500 Retry)
            </button>
            <button class="btn btn-red" onclick="quickEnqueue('webhook:send', {url:'https://httpbin.org/post', method:'POST', simulate_code:404})">
                Webhook (404 Skip)
            </button>
            <button class="btn btn-green" onclick="batchNotifications()">
                Batch Notifications (x5)
            </button>
        </div>
    </div>
</div>

<script>
const payloads = {
    'email:deliver': { to: 'user@example.com', subject: 'Hello World', body: 'This is a test email.' },
    'image:resize': { url: 'https://example.com/image.jpg', width: 800, height: 600 },
    'report:generate': { report_type: 'daily-summary', start_date: '2025-01-01', end_date: '2025-01-07' },
    'webhook:send': { url: 'https://httpbin.org/post', method: 'POST', simulate_code: 200 },
    'notification:send': { user_id: 1, message: 'Hello!', channel: 'email' }
};

function updatePayload() {
    const type = document.getElementById('task-type').value;
    document.getElementById('task-payload').value = JSON.stringify(payloads[type] || {}, null, 2);
}
updatePayload();

function showResult(data, isError) {
    const el = document.getElementById('enqueue-result');
    el.className = isError ? 'error' : '';
    el.textContent = JSON.stringify(data, null, 2);
}

async function enqueueTask() {
    const form = document.getElementById('enqueue-form');
    const body = {
        type: document.getElementById('task-type').value,
        payload: JSON.parse(document.getElementById('task-payload').value),
        queue: document.getElementById('task-queue').value,
        delay_seconds: parseInt(document.getElementById('task-delay').value) || 0,
        max_retry: parseInt(document.getElementById('task-retry').value) || 0,
        unique_ttl_seconds: parseInt(document.getElementById('task-unique').value) || 0
    };
    try {
        const resp = await fetch('/api/tasks/enqueue', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(body)
        });
        const data = await resp.json();
        showResult(data, !resp.ok);
    } catch (e) {
        showResult({ error: e.message }, true);
    }
}

async function quickEnqueue(type, payload) {
    try {
        const resp = await fetch('/api/tasks/enqueue', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ type, payload })
        });
        const data = await resp.json();
        showResult(data, !resp.ok);
    } catch (e) {
        showResult({ error: e.message }, true);
    }
}

async function batchNotifications() {
    const results = [];
    for (let i = 1; i <= 5; i++) {
        try {
            const resp = await fetch('/api/tasks/enqueue', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    type: 'notification:send',
                    payload: { user_id: 42, message: 'Notification #' + i, channel: 'push' }
                })
            });
            results.push(await resp.json());
        } catch (e) {
            results.push({ error: e.message });
        }
    }
    showResult({ message: '5 notifications enqueued for user 42 (group aggregation)', results }, false);
}
</script>
</body>
</html>`
