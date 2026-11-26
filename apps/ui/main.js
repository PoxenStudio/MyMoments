const { invoke } = window.__TAURI__.core;

const form = document.getElementById('form');
const input = document.getElementById('server');
const submit = document.getElementById('submit');
const error = document.getElementById('error');

function showError(message) {
  error.textContent = message || '连接失败，请重试';
  error.hidden = false;
}

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
