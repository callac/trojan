/**
 * Trojan 管理系统扩展模块 (Ext Plugin)
 * 采用原生 DOM 与 Element Plus 风格无缝集成：
 * 1. 在 /dashboard 仪表盘提供 BBR 状态展示与一键激活
 * 2. 在 /trojan 服务页面提供 主端口修改 & 附加多端口透明重定向设置
 */
(function () {
  function getCookie(name) {
    const value = `; ${document.cookie}`;
    const parts = value.split(`; ${name}=`);
    if (parts.length === 2) return parts.pop().split(';').shift();
    return '';
  }

  function getAuthHeaders() {
    const token = getCookie('jwt') || (window.Vuex && window.Vuex.Store ? '' : '');
    const headers = {};
    if (token) {
      headers['Authorization'] = 'Bearer ' + token;
    }
    return headers;
  }

  // 1. Dashboard 注入 BBR 状态卡片
  async function initBBRCard() {
    const hash = window.location.hash;
    if (!hash.includes('/dashboard')) return;
    if (document.getElementById('ext-bbr-card')) return;

    // 寻找放卡片的 row 容器
    const homeCards = document.querySelectorAll('.home-card');
    if (!homeCards || homeCards.length < 2) return;

    const parentCol = homeCards[homeCards.length - 1].parentElement;
    if (!parentCol || !parentCol.parentElement) return;

    // 获取 BBR 状态
    try {
      const resp = await axios.get('/trojan/bbr', { headers: getAuthHeaders() });
      const bbrData = resp.data && resp.data.Data ? resp.data.Data : { enabled: false, algo: 'unknown' };

      if (document.getElementById('ext-bbr-card')) return;

      const cardWrapper = document.createElement('div');
      cardWrapper.id = 'ext-bbr-card';
      cardWrapper.className = 'el-col el-col-24 el-col-sm-24 el-col-md-12';
      cardWrapper.style.marginTop = '10px';

      cardWrapper.innerHTML = `
        <div class="el-card home-card is-hover-shadow" style="padding: 10px 15px;">
          <div class="el-row" style="display: flex; align-items: center; justify-content: space-between;">
            <div style="font-weight: bold; color: #303133; font-size: 14px;">
              BBR 加速状态:
            </div>
            <div id="ext-bbr-action" style="display: flex; align-items: center; gap: 10px;">
              ${
                bbrData.enabled
                  ? `<span class="el-tag el-tag--success el-tag--dark el-tag--default">已开启 (${bbrData.algo})</span>`
                  : `<span class="el-tag el-tag--warning el-tag--default">未开启 (${bbrData.algo})</span>
                     <button id="ext-enable-bbr-btn" class="el-button el-button--primary el-button--small">一键开启</button>`
              }
            </div>
          </div>
        </div>
      `;

      parentCol.parentElement.appendChild(cardWrapper);

      const btn = document.getElementById('ext-enable-bbr-btn');
      if (btn) {
        btn.addEventListener('click', async () => {
          btn.disabled = true;
          btn.innerText = '开启中...';
          try {
            const postResp = await axios.post('/trojan/bbr', {}, { headers: getAuthHeaders() });
            if (postResp.data && postResp.data.Msg === 'success') {
              if (window.ElementPlus && window.ElementPlus.ElMessage) {
                window.ElementPlus.ElMessage.success('BBR 拥塞控制加速已成功开启！');
              }
              const actionBox = document.getElementById('ext-bbr-action');
              if (actionBox) {
                actionBox.innerHTML = '<span class="el-tag el-tag--success el-tag--dark el-tag--default">已开启 (bbr)</span>';
              }
            } else {
              throw new Error(postResp.data.Msg || '开启失败');
            }
          } catch (err) {
            btn.disabled = false;
            btn.innerText = '重试';
            if (window.ElementPlus && window.ElementPlus.ElMessage) {
              window.ElementPlus.ElMessage.error(err.message || '开启 BBR 失败');
            }
          }
        });
      }
    } catch (e) {
      console.warn('获取 BBR 状态失败:', e);
    }
  }

  // 2. Trojan 页面注入“端口设置”模态对话框
  async function initPortManager() {
    const hash = window.location.hash;
    if (!hash.includes('/trojan')) return;
    if (document.getElementById('ext-port-btn')) return;

    // 寻找操作按钮行
    const btnGroup = document.querySelector('.el-button-group');
    if (!btnGroup) return;

    // 插入端口设置按钮
    const portBtn = document.createElement('button');
    portBtn.id = 'ext-port-btn';
    portBtn.type = 'button';
    portBtn.className = 'el-button el-button--warning';
    portBtn.style.marginLeft = '10px';
    portBtn.innerHTML = '<span>端口设置</span>';
    btnGroup.parentElement.appendChild(portBtn);

    portBtn.addEventListener('click', () => {
      openPortDialog();
    });
  }

  // 渲染并打开端口设置对话框
  async function openPortDialog() {
    let existingDialog = document.getElementById('ext-port-dialog-mask');
    if (existingDialog) {
      existingDialog.remove();
    }

    // 获取当前端口配置
    let currentMainPort = 443;
    let currentMultiPorts = [];
    try {
      const resp = await axios.get('/trojan/port', { headers: getAuthHeaders() });
      if (resp.data && resp.data.Data) {
        currentMainPort = resp.data.Data.mainPort || 443;
        currentMultiPorts = resp.data.Data.multiPorts || [];
      }
    } catch (e) {
      console.warn('获取端口配置失败:', e);
    }

    const mask = document.createElement('div');
    mask.id = 'ext-port-dialog-mask';
    mask.style.cssText = `
      position: fixed; top: 0; left: 0; width: 100vw; height: 100vh;
      background: rgba(0,0,0,0.5); z-index: 3000; display: flex;
      align-items: center; justify-content: center;
    `;

    mask.innerHTML = `
      <div class="el-overlay-dialog" style="width: 520px; max-width: 90vw; background: #fff; border-radius: 8px; box-shadow: 0 4px 16px rgba(0,0,0,0.2); overflow: hidden; font-family: sans-serif;">
        <div style="padding: 16px 20px; border-bottom: 1px solid #ebeef5; display: flex; justify-content: space-between; align-items: center;">
          <span style="font-size: 16px; font-weight: bold; color: #303133;">代理端口管理 (主端口 & 附加多端口)</span>
          <span id="ext-port-close" style="cursor: pointer; font-size: 18px; color: #909399;">✕</span>
        </div>
        <div style="padding: 20px;">
          <div style="margin-bottom: 16px;">
            <label style="display: block; font-size: 14px; font-weight: bold; color: #606266; margin-bottom: 6px;">
              Trojan 主监听端口:
            </label>
            <input id="ext-input-main-port" type="number" min="1" max="65535" value="${currentMainPort}"
              class="el-input__inner" style="width: 100%; height: 38px; border: 1px solid #dcdfe6; border-radius: 4px; padding: 0 12px; box-sizing: border-box;" />
            <div style="font-size: 12px; color: #909399; margin-top: 4px;">
              Trojan 服务核心监听端口（默认 443，需确保未被 Nginx/Apache 占用）。
            </div>
          </div>
          <div style="margin-bottom: 16px;">
            <label style="display: block; font-size: 14px; font-weight: bold; color: #606266; margin-bottom: 6px;">
              附加多对外端口 (逗号分隔):
            </label>
            <input id="ext-input-multi-ports" type="text" value="${currentMultiPorts.join(', ')}"
              placeholder="例如: 8443, 2083, 2087"
              class="el-input__inner" style="width: 100%; height: 38px; border: 1px solid #dcdfe6; border-radius: 4px; padding: 0 12px; box-sizing: border-box;" />
            <div style="font-size: 12px; color: #909399; margin-top: 4px;">
              基于 Linux 内核 iptables PREROUTING 透明重定向，外部连接这些端口将全量转发到主端口，实现多端口同时可用。
            </div>
          </div>
        </div>
        <div style="padding: 12px 20px; background: #f5f7fa; border-top: 1px solid #ebeef5; text-align: right;">
          <button id="ext-port-cancel-btn" class="el-button el-button--default" style="margin-right: 10px;">取消</button>
          <button id="ext-port-save-btn" class="el-button el-button--primary">保存并应用</button>
        </div>
      </div>
    `;

    document.body.appendChild(mask);

    document.getElementById('ext-port-close').onclick = () => mask.remove();
    document.getElementById('ext-port-cancel-btn').onclick = () => mask.remove();

    document.getElementById('ext-port-save-btn').onclick = async () => {
      const saveBtn = document.getElementById('ext-port-save-btn');
      const mainPortVal = parseInt(document.getElementById('ext-input-main-port').value.trim());
      const multiPortsVal = document.getElementById('ext-input-multi-ports').value.trim();

      if (isNaN(mainPortVal) || mainPortVal < 1 || mainPortVal > 65535) {
        if (window.ElementPlus && window.ElementPlus.ElMessage) {
          window.ElementPlus.ElMessage.error('主端口必须为 1~65535 之间的有效端口！');
        }
        return;
      }

      saveBtn.disabled = true;
      saveBtn.innerText = '正在保存...';

      try {
        // 1. 保存主端口
        const pData = new FormData();
        pData.set('port', mainPortVal);
        const pResp = await axios.post('/trojan/port', pData, { headers: getAuthHeaders() });
        if (pResp.data.Msg !== 'success') {
          throw new Error('保存主端口失败: ' + pResp.data.Msg);
        }

        // 2. 保存附加多端口
        const mData = new FormData();
        mData.set('ports', multiPortsVal);
        const mResp = await axios.post('/trojan/multiport', mData, { headers: getAuthHeaders() });
        if (mResp.data.Msg !== 'success') {
          throw new Error('保存附加多端口失败: ' + mResp.data.Msg);
        }

        mask.remove();

        if (window.ElementPlus && window.ElementPlus.ElMessageBox) {
          window.ElementPlus.ElMessageBox.confirm(
            '端口配置已成功保存！是否立即重启 Trojan 服务以使新端口生效？',
            '提示',
            {
              confirmButtonText: '立即重启',
              cancelButtonText: '稍后手动重启',
              type: 'success',
            }
          ).then(async () => {
            try {
              await axios.post('/trojan/restart', {}, { headers: getAuthHeaders() });
              window.ElementPlus.ElMessage.success('Trojan 服务已重启！');
            } catch (err) {
              window.ElementPlus.ElMessage.warning('重启指令已发送');
            }
          }).catch(() => {});
        } else {
          alert('端口保存成功！请在页面中点击“重启”服务以生效。');
        }
      } catch (err) {
        saveBtn.disabled = false;
        saveBtn.innerText = '保存并应用';
        if (window.ElementPlus && window.ElementPlus.ElMessage) {
          window.ElementPlus.ElMessage.error(err.message || '保存失败');
        } else {
          alert(err.message || '保存失败');
        }
      }
    };
  }

  // 循环/事件检查当前路由并尝试挂载
  function checkRouteAndMount() {
    initBBRCard();
    initPortManager();
  }

  window.addEventListener('hashchange', () => {
    setTimeout(checkRouteAndMount, 300);
    setTimeout(checkRouteAndMount, 1000);
  });

  setInterval(checkRouteAndMount, 1500);

  // 初始化首次挂载
  setTimeout(checkRouteAndMount, 500);
  setTimeout(checkRouteAndMount, 1500);
})();
