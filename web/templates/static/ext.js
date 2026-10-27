/**
 * Trojan 管理系统扩展模块 (Ext Plugin)
 * 采用原生 DOM 与 Element Plus 标准组件样式无缝集成：
 * 1. 在 /dashboard 仪表盘提供 BBR 状态展示/一键激活 & 代理端口概览
 * 2. 在 /trojan 服务页面按钮组中原生集成“端口设置”模态对话框
 */
(function () {
  function getCookie(name) {
    const value = `; ${document.cookie}`;
    const parts = value.split(`; ${name}=`);
    if (parts.length === 2) return parts.pop().split(';').shift();
    return '';
  }

  function getAuthHeaders() {
    const token = getCookie('jwt');
    const headers = {};
    if (token) {
      headers['Authorization'] = 'Bearer ' + token;
    }
    return headers;
  }

  // 1. Dashboard 注入 BBR 状态与代理端口整行卡片
  async function initBBRCard() {
    const hash = window.location.hash;
    if (!hash.includes('/dashboard')) return;
    if (document.getElementById('ext-dashboard-row')) return;

    // 寻找 Dashboard 现有的 home-card 列表
    const homeCards = document.querySelectorAll('.home-card');
    if (!homeCards || homeCards.length < 2) return;

    // 找到放置卡片的主容器
    const lastCardCol = homeCards[homeCards.length - 1].parentElement;
    if (!lastCardCol || !lastCardCol.parentElement) return;
    const parentContainer = lastCardCol.parentElement.parentElement;
    if (!parentContainer) return;

    try {
      // 并行获取 BBR 状态与当前端口信息
      const [bbrResp, portResp] = await Promise.all([
        axios.get('/trojan/bbr', { headers: getAuthHeaders() }).catch(() => null),
        axios.get('/trojan/port', { headers: getAuthHeaders() }).catch(() => null),
      ]);

      const bbrData = bbrResp && bbrResp.data && bbrResp.data.Data ? bbrResp.data.Data : { enabled: false, algo: 'unknown' };
      const portData = portResp && portResp.data && portResp.data.Data ? portResp.data.Data : { mainPort: 443, multiPorts: [] };

      if (document.getElementById('ext-dashboard-row')) return;

      const rowWrapper = document.createElement('div');
      rowWrapper.id = 'ext-dashboard-row';
      rowWrapper.className = 'el-row';
      rowWrapper.style.cssText = 'margin-top: 10px;';

      const multiPortDesc = portData.multiPorts && portData.multiPorts.length > 0
        ? ` <span style="color:#909399;font-size:12px;">(+${portData.multiPorts.join(',')})</span>`
        : '';

      rowWrapper.innerHTML = `
        <div class="el-col el-col-24 el-col-sm-24 el-col-md-12">
          <div class="el-card home-card is-hover-shadow">
            <div class="el-card__body">
              <div class="el-row" style="display:flex;align-items:center;">
                <div class="el-col el-col-10">
                  <b>BBR 加速:</b>
                </div>
                <div id="ext-bbr-action" class="el-col el-col-12" style="padding-top:1px;display:flex;align-items:center;gap:8px;">
                  ${
                    bbrData.enabled
                      ? `<span class="el-tag el-tag--success el-tag--light el-tag--small">已开启 (${bbrData.algo})</span>`
                      : `<span class="el-tag el-tag--warning el-tag--light el-tag--small">未开启 (${bbrData.algo})</span>
                         <button id="ext-enable-bbr-btn" class="el-button el-button--primary el-button--small" style="padding:4px 8px;height:24px;">一键开启</button>`
                  }
                </div>
              </div>
            </div>
          </div>
        </div>
        <div class="el-col el-col-24 el-col-sm-24 el-col-md-12">
          <div class="el-card home-card is-hover-shadow">
            <div class="el-card__body">
              <div class="el-row" style="display:flex;align-items:center;">
                <div class="el-col el-col-10">
                  <b>代理端口:</b>
                </div>
                <div class="el-col el-col-12" style="padding-top:1px;">
                  <span style="font-weight:600;color:#409EFF;">${portData.mainPort || 443}</span>${multiPortDesc}
                </div>
              </div>
            </div>
          </div>
        </div>
      `;

      // 插入到现有卡片行下方
      lastCardCol.parentElement.parentElement.insertBefore(rowWrapper, lastCardCol.parentElement.nextSibling);

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
                actionBox.innerHTML = '<span class="el-tag el-tag--success el-tag--light el-tag--small">已开启 (bbr)</span>';
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
      console.warn('获取 Dashboard 扩展信息失败:', e);
    }
  }

  // 2. 在 /trojan 页面顶部按钮组原生嵌入“端口设置”按钮
  async function initPortManager() {
    const hash = window.location.hash;
    if (!hash.includes('/trojan')) return;
    if (document.getElementById('ext-port-btn')) return;

    // 寻找原版 el-button-group 操作按钮组
    const btnGroup = document.querySelector('.el-button-group');
    if (!btnGroup) return;

    // 创建原生的 el-button 并追加进 button-group，浑然一体
    const portBtn = document.createElement('button');
    portBtn.id = 'ext-port-btn';
    portBtn.type = 'button';
    portBtn.className = 'el-button el-button--primary';
    portBtn.innerHTML = '<span>端口设置</span>';
    btnGroup.appendChild(portBtn);

    portBtn.addEventListener('click', () => {
      openPortDialog();
    });
  }

  // 渲染并打开端口管理对话框
  async function openPortDialog() {
    let existingDialog = document.getElementById('ext-port-dialog-mask');
    if (existingDialog) {
      existingDialog.remove();
    }

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
      <div class="el-overlay-dialog" style="width: 520px; max-width: 90vw; background: #fff; border-radius: 8px; box-shadow: 0 8px 24px rgba(0,0,0,0.15); overflow: hidden; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif;">
        <div style="padding: 16px 20px; border-bottom: 1px solid #ebeef5; display: flex; justify-content: space-between; align-items: center;">
          <span style="font-size: 16px; font-weight: 600; color: #303133;">代理端口管理 (主端口 & 附加多端口)</span>
          <span id="ext-port-close" style="cursor: pointer; font-size: 18px; color: #909399;">✕</span>
        </div>
        <div style="padding: 20px;">
          <div style="background: #fdf6ec; border: 1px solid #faecd8; border-radius: 4px; padding: 10px 14px; margin-bottom: 16px; font-size: 13px; color: #e6a23c; line-height: 1.5;">
            <b>特别提示：</b>若修改主监听端口（如 444）或配置附加端口，除系统自动放行外，<b>请务必确认已在云服务器控制台（如腾讯云/阿里云/AWS/甲骨文）的安全组规则中放行对应端口的 TCP/UDP 入站</b>，否则外部网络会被安全组拦截导致无法建立连接！
          </div>
          <div style="margin-bottom: 18px;">
            <label style="display: block; font-size: 14px; font-weight: 600; color: #606266; margin-bottom: 8px;">
              Trojan 主监听端口:
            </label>
            <div class="el-input el-input--default">
              <div class="el-input__wrapper" style="box-shadow: 0 0 0 1px #dcdfe6 inset; border-radius: 4px; padding: 1px 11px;">
                <input id="ext-input-main-port" type="number" min="1" max="65535" value="${currentMainPort}"
                  class="el-input__inner" style="height: 36px; border: none; outline: none; width: 100%; color: #606266; font-size: 14px;" />
              </div>
            </div>
            <div style="font-size: 12px; color: #909399; margin-top: 5px;">
              Trojan 服务核心监听端口（默认 443，需确保未被 Nginx/Apache 等其它软件占用）。
            </div>
          </div>
          <div style="margin-bottom: 18px;">
            <label style="display: block; font-size: 14px; font-weight: 600; color: #606266; margin-bottom: 8px;">
              附加多对外端口 (逗号分隔):
            </label>
            <div class="el-input el-input--default">
              <div class="el-input__wrapper" style="box-shadow: 0 0 0 1px #dcdfe6 inset; border-radius: 4px; padding: 1px 11px;">
                <input id="ext-input-multi-ports" type="text" value="${currentMultiPorts.join(', ')}"
                  placeholder="例如: 8443, 2083, 2087"
                  class="el-input__inner" style="height: 36px; border: none; outline: none; width: 100%; color: #606266; font-size: 14px;" />
              </div>
            </div>
            <div style="font-size: 12px; color: #909399; margin-top: 5px;">
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

  // 路由监控与动态挂载
  function checkRouteAndMount() {
    initBBRCard();
    initPortManager();
  }

  window.addEventListener('hashchange', () => {
    setTimeout(checkRouteAndMount, 300);
    setTimeout(checkRouteAndMount, 1000);
  });

  setInterval(checkRouteAndMount, 1500);

  setTimeout(checkRouteAndMount, 500);
  setTimeout(checkRouteAndMount, 1500);
})();
