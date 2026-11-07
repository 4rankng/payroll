const http = require('http');
const postJson = (path, payload) => new Promise((resolve, reject) => {
  const data = JSON.stringify(payload);
  const req = http.request({ host: 'localhost', port: 8081, path, method: 'POST',
    headers: { 'Content-Type': 'application/json', 'Content-Length': Buffer.byteLength(data) } },
    (res) => { let buf = ''; res.on('data', (c) => (buf += c));
      res.on('end', () => resolve({ status: res.statusCode, body: buf })); });
  req.on('error', reject);
  req.write(data);
  req.end();
});
(async () => {
  const { body } = await postJson('/api/v1/auth/login', { username: 'frankng', password: 'Admin123' });
  const parsed = JSON.parse(body);
  const d = parsed.data || {};
  console.log('top-keys:', Object.keys(parsed).join(','));
  console.log('data-keys:', Object.keys(d).join(','));
})();
