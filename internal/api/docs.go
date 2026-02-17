package api

import "net/http"

func HandleDocs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(docsHTML))
}

const docsHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>WA MVP API Docs</title>
  <style>
    :root {
      --bg: #f7fbf7;
      --card: #ffffff;
      --text: #0f1a12;
      --muted: #5a6b5f;
      --accent: #1f8b4c;
      --accent-soft: #e5f5ec;
      --border: #dbe6df;
      --code: #f1f6f2;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      background: var(--bg);
      color: var(--text);
      font-family: ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, Helvetica, Arial, sans-serif;
      line-height: 1.5;
    }
    .wrap {
      max-width: 1180px;
      margin: 36px auto;
      padding: 0 20px 60px;
    }
    h1, h2, h3 { margin: 0 0 12px; }
    h1 { font-size: 34px; }
    h2 { font-size: 20px; }
    p { color: var(--muted); margin: 6px 0 0; }
    .grid {
      display: grid;
      grid-template-columns: 280px 1fr;
      gap: 18px;
      margin-top: 18px;
    }
    .side {
      background: var(--card);
      border: 1px solid var(--border);
      border-radius: 12px;
      padding: 14px;
      position: sticky;
      top: 16px;
      height: fit-content;
    }
    .side h2 {
      color: var(--accent);
    }
    .steps {
      list-style: none;
      padding: 0;
      margin: 10px 0 0;
    }
    .steps li {
      padding: 10px 10px;
      border-radius: 10px;
      border: 1px solid transparent;
      margin-bottom: 8px;
      background: var(--accent-soft);
    }
    .steps a {
      display: block;
      color: inherit;
      text-decoration: none;
    }
    .steps span {
      display: inline-block;
      font-weight: 600;
      margin-right: 8px;
      color: var(--accent);
    }
    .section {
      scroll-margin-top: 80px;
    }
    .main {
      display: grid;
      gap: 14px;
    }
    .card {
      background: var(--card);
      border: 1px solid var(--border);
      border-radius: 12px;
      padding: 16px 18px;
    }
    code, pre {
      font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, Liberation Mono, monospace;
    }
    pre {
      background: var(--code);
      border: 1px solid var(--border);
      border-radius: 10px;
      padding: 12px;
      overflow-x: auto;
      color: #0b1b11;
      margin: 8px 0 0;
    }
    .tag {
      display: inline-block;
      font-size: 12px;
      padding: 2px 8px;
      border-radius: 999px;
      background: var(--accent);
      color: white;
      margin-left: 6px;
      vertical-align: middle;
    }
    .warn {
      color: #8b6a1f;
    }
    @media (max-width: 900px) {
      .grid { grid-template-columns: 1fr; }
      .side { position: static; }
    }
  </style>
</head>
<body>
  <div class="wrap">
    <h1>WA MVP API Documentation</h1>
    <p>Base URL: <code>http://localhost:9090</code></p>

    <div class="grid">
      <aside class="side">
        <h2>Steps</h2>
        <ul class="steps">
          <li><a href="#create"><span>1</span>Create a session</a></li>
          <li><a href="#qr"><span>2</span>Fetch QR and scan</a></li>
          <li><a href="#status"><span>3</span>Check status</a></li>
          <li><a href="#send"><span>4</span>Send a message</a></li>
          <li><a href="#receive"><span>5</span>Receive messages (polling)</a></li>
        </ul>
      </aside>

      <main class="main">
        <div class="card section" id="auth">
          <h2>Authentication</h2>
          <p>Session-specific endpoints require a Bearer token returned by <code>POST /sessions</code>.</p>
          <pre>Authorization: Bearer &lt;YOUR_TOKEN&gt;</pre>
        </div>

        <div class="card section" id="create">
          <h2>Create Session <span class="tag">POST</span></h2>
          <p>Creates a new WhatsApp session. The response includes a bearer token that identifies this session. Store it securely and send it in the <code>Authorization</code> header for all session-specific calls.</p>
          <pre>curl -X POST http://localhost:9090/sessions</pre>
          <p>Response:</p>
          <pre>{"token":"YOUR_TOKEN"}</pre>
        </div>

        <div class="card section" id="list">
          <h2>List Sessions <span class="tag">GET</span></h2>
          <p>Lists all sessions (for admin/debug). No auth required. Useful to see connection state and JID.</p>
          <pre>curl http://localhost:9090/sessions</pre>
          <p>Response:</p>
          <pre>[{"id":"abc123","connected":true,"jid":"9198xxx@s.whatsapp.net"}]</pre>
        </div>

        <div class="card section" id="qr">
          <h2>Get QR <span class="tag">GET</span></h2>
          <p>Returns a base64 PNG QR for the session. Open the URL in a browser and render the image, or decode the base64 to show it in your UI.</p>
          <pre>curl http://localhost:9090/session/qr \
  -H "Authorization: Bearer YOUR_TOKEN"</pre>
          <p>Response:</p>
          <pre>{"qr":"data:image/png;base64,..."}</pre>
        </div>

        <div class="card section" id="status">
          <h2>Get Status <span class="tag">GET</span></h2>
          <p>Returns login and connection status for the session. Use this after scanning the QR to confirm the session is active.</p>
          <pre>curl http://localhost:9090/session/status \
  -H "Authorization: Bearer YOUR_TOKEN"</pre>
          <p>Response:</p>
          <pre>{"logged_in":true,"connected":true,"jid":"9198xxx@s.whatsapp.net"}</pre>
        </div>

        <div class="card section" id="send">
          <h2>Send Message <span class="tag">POST</span></h2>
          <p>Sends a text message from the session. Phone should be in international format without <code>+</code>.</p>
          <pre>curl -X POST http://localhost:9090/session/send \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"phone\":\"919999999999\",\"message\":\"hello\"}"</pre>
          <p>Response:</p>
          <pre>{"status":"sent"}</pre>
        </div>

        <div class="card section" id="receive">
          <h2>Receive Messages (Polling) <span class="tag">GET</span></h2>
          <p>Returns and clears queued incoming messages for the session. Call this endpoint periodically to fetch new messages.</p>
          <pre>curl http://localhost:9090/session/receive \
  -H "Authorization: Bearer YOUR_TOKEN"</pre>
          <p>Optional limit:</p>
          <pre>curl "http://localhost:9090/session/receive?limit=50" \
  -H "Authorization: Bearer YOUR_TOKEN"</pre>
          <p>Response:</p>
          <pre>{"messages":[{"from":"919xxxxxxx","name":"Contact Name","message":"hello","timestamp":1700000000}]}</pre>
          <p class="warn">Notes: Only text messages are captured. Media is ignored. Messages are stored in memory and removed when fetched.</p>
        </div>
      </main>
    </div>
  </div>
</body>
</html>`
