const { invoke } = window.__TAURI__.core;

const form = document.getElementById('form');
const input = document.getElementById('server');
const submit = document.getElementById('submit');
const error = document.getElementById('error');
const recent = document.getElementById('recent');

function showError(message) {
  error.textContent = message || '连接失败，请重试';
  error.hidden = false;
}

async function loadRecent() {
  try {
    const urls = await invoke('recent_servers');
    if (!Array.isArray(urls) || urls.length === 0) return;
    recent.innerHTML = '';
    for (const url of urls) {
      const option = document.createElement('option');
      option.value = url;
      recent.appendChild(option);
    }
    if (!input.value) {
      input.value = urls[0];
      input.select();
    }
  } catch {
    // 历史地址读取失败不影响手动输入
  }
}

loadRecent();

form.addEventListener('submit', async (event) => {
  event.preventDefault();
  error.hidden = true;
  const url = input.value.trim();
  if (!url) {
    showError('请输入服务器地址');
    return;
  }
  submit.disabled = true;
  try {
    await invoke('save_server_url', { url });
  } catch (err) {
    showError(typeof err === 'string' ? err : err?.message);
    submit.disabled = false;
  }
});
