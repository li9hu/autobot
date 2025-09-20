// Bark管理页面JavaScript

// 全局变量
let currentServerPage = 1;
let currentDevicePage = 1;
let currentEditingServerId = null;
let currentEditingDeviceId = null;

// 页面初始化
document.addEventListener('DOMContentLoaded', function() {
    initializeTabs();
    loadServers();
    loadDevices();
    setupEventListeners();
});

// 初始化选项卡
function initializeTabs() {
    const serversTab = document.getElementById('servers-tab');
    const devicesTab = document.getElementById('devices-tab');
    const serversContent = document.getElementById('servers-content');
    const devicesContent = document.getElementById('devices-content');
    
    // 设置初始状态
    updateTabState(serversTab, true);
    updateTabState(devicesTab, false);
    
    serversTab.addEventListener('click', function() {
        updateTabState(serversTab, true);
        updateTabState(devicesTab, false);
        serversContent.classList.remove('hidden');
        devicesContent.classList.add('hidden');
    });
    
    devicesTab.addEventListener('click', function() {
        updateTabState(serversTab, false);
        updateTabState(devicesTab, true);
        serversContent.classList.add('hidden');
        devicesContent.classList.remove('hidden');
    });
}

// 更新选项卡状态
function updateTabState(tab, isActive) {
    if (isActive) {
        tab.classList.add('text-blue-600', 'border-blue-600');
        tab.classList.remove('text-slate-500', 'border-transparent', 'hover:text-slate-700', 'hover:border-slate-300');
    } else {
        tab.classList.remove('text-blue-600', 'border-blue-600');
        tab.classList.add('text-slate-500', 'border-transparent', 'hover:text-slate-700', 'hover:border-slate-300');
    }
}

// 设置事件监听器
function setupEventListeners() {
    // 服务器表单提交
    document.getElementById('server-form').addEventListener('submit', function(e) {
        e.preventDefault();
        saveServer();
    });
    
    // 设备表单提交
    document.getElementById('device-form').addEventListener('submit', function(e) {
        e.preventDefault();
        saveDevice();
    });
    
    // 模态框点击外部关闭
    window.addEventListener('click', function(e) {
        const serverModal = document.getElementById('server-modal');
        const deviceModal = document.getElementById('device-modal');
        
        if (e.target === serverModal) {
            closeServerModal();
        }
        if (e.target === deviceModal) {
            closeDeviceModal();
        }
    });
}

// ===== 服务器管理 =====

// 加载服务器列表
async function loadServers(page = 1) {
    currentServerPage = page;
    const loading = document.getElementById('servers-loading');
    const tableContainer = document.getElementById('servers-table-container');
    
    loading.classList.remove('hidden');
    tableContainer.classList.add('hidden');
    
    try {
        const response = await fetch(`/api/bark/servers?page=${page}&limit=10`);
        const data = await response.json();
        
        if (response.ok) {
            renderServersTable(data.servers || []);
            renderServersPagination(data.total || 0, data.page || 1, data.limit || 10);
        } else {
            showAlert('加载服务器列表失败: ' + (data.error || '未知错误'), 'error');
        }
    } catch (error) {
        console.error('Error loading servers:', error);
        showAlert('加载服务器列表失败: ' + error.message, 'error');
    } finally {
        loading.classList.add('hidden');
        tableContainer.classList.remove('hidden');
    }
}

// 渲染服务器表格
function renderServersTable(servers) {
    const tbody = document.getElementById('servers-tbody');
    
    if (servers.length === 0) {
        tbody.innerHTML = `
            <tr>
                <td colspan="7" class="px-6 py-12 text-center">
                    <div class="flex flex-col items-center">
                        <i data-lucide="server" class="w-12 h-12 text-slate-300 mb-4"></i>
                        <h3 class="text-sm font-medium text-slate-900 mb-1">暂无服务器配置</h3>
                        <p class="text-sm text-slate-500 mb-4">开始添加您的第一个 Bark 服务器</p>
                        <button onclick="showServerModal()" class="inline-flex items-center px-4 py-2 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-700 transition-colors">
                            <i data-lucide="plus" class="w-4 h-4 mr-2"></i>
                            添加服务器
                        </button>
                    </div>
                </td>
            </tr>
        `;
        // 重新初始化图标
        lucide.createIcons();
        return;
    }
    
    tbody.innerHTML = servers.map(server => `
        <tr class="hover:bg-slate-50">
            <td class="px-6 py-4 whitespace-nowrap">
                <div class="flex items-center">
                    <div class="text-sm font-medium text-slate-900">${escapeHtml(server.name)}</div>
                    ${server.is_default ? '<span class="ml-2 inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-yellow-100 text-yellow-800">默认</span>' : ''}
                </div>
            </td>
            <td class="px-6 py-4 whitespace-nowrap">
                <code class="text-sm text-slate-600 bg-slate-100 px-2 py-1 rounded">${escapeHtml(server.url)}</code>
            </td>
            <td class="px-6 py-4">
                <div class="text-sm text-slate-600 max-w-xs truncate">${escapeHtml(server.description || '-')}</div>
            </td>
            <td class="px-6 py-4 whitespace-nowrap">
                <span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${server.status === 'active' ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'}">
                    ${server.status === 'active' ? '启用' : '禁用'}
                </span>
            </td>
            <td class="px-6 py-4 whitespace-nowrap text-sm text-slate-600">
                ${server.is_default ? '是' : '否'}
            </td>
            <td class="px-6 py-4 whitespace-nowrap text-sm text-slate-600">
                ${formatDateTime(server.created_at)}
            </td>
            <td class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                <div class="flex items-center justify-end space-x-2">
                    <button onclick="editServer(${server.id})" class="text-blue-600 hover:text-blue-900">编辑</button>
                    <button onclick="deleteServer(${server.id}, '${escapeHtml(server.name)}')" class="text-red-600 hover:text-red-900">删除</button>
                </div>
            </td>
        </tr>
    `).join('');
}

// 渲染服务器分页
function renderServersPagination(total, page, limit) {
    const totalPages = Math.ceil(total / limit);
    const pagination = document.getElementById('servers-pagination');
    
    if (totalPages <= 1) {
        pagination.innerHTML = '';
        return;
    }
    
    let html = `
        <button onclick="loadServers(${page - 1})" ${page <= 1 ? 'disabled' : ''} 
                class="px-3 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-300 rounded-lg hover:bg-slate-50 disabled:opacity-50 disabled:cursor-not-allowed">
            上一页
        </button>
        <span class="px-3 py-2 text-sm text-slate-600">第 ${page} 页，共 ${totalPages} 页</span>
        <button onclick="loadServers(${page + 1})" ${page >= totalPages ? 'disabled' : ''} 
                class="px-3 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-300 rounded-lg hover:bg-slate-50 disabled:opacity-50 disabled:cursor-not-allowed">
            下一页
        </button>
    `;
    
    pagination.innerHTML = html;
}

// 显示服务器模态框
function showServerModal(serverId = null) {
    currentEditingServerId = serverId;
    const modal = document.getElementById('server-modal');
    const title = document.getElementById('server-modal-title');
    const form = document.getElementById('server-form');
    
    // 重置表单
    form.reset();
    
    if (serverId) {
        title.textContent = '编辑服务器';
        loadServerData(serverId);
    } else {
        title.textContent = '添加服务器';
    }
    
    modal.classList.remove('hidden');
}

// 加载服务器数据
async function loadServerData(serverId) {
    try {
        const response = await fetch(`/api/bark/servers/${serverId}`);
        const server = await response.json();
        
        if (response.ok) {
            document.getElementById('server-name').value = server.name || '';
            document.getElementById('server-url').value = server.url || '';
            document.getElementById('server-description').value = server.description || '';
            document.getElementById('server-is-default').checked = server.is_default || false;
        } else {
            showAlert('加载服务器数据失败: ' + (server.error || '未知错误'), 'error');
        }
    } catch (error) {
        console.error('Error loading server data:', error);
        showAlert('加载服务器数据失败: ' + error.message, 'error');
    }
}

// 保存服务器
async function saveServer() {
    const name = document.getElementById('server-name').value.trim();
    const url = document.getElementById('server-url').value.trim();
    const description = document.getElementById('server-description').value.trim();
    const isDefault = document.getElementById('server-is-default').checked;
    
    if (!name || !url) {
        showAlert('请填写必填字段', 'error');
        return;
    }
    
    const data = {
        name,
        url,
        description,
        is_default: isDefault
    };
    
    try {
        let response;
        if (currentEditingServerId) {
            response = await fetch(`/api/bark/servers/${currentEditingServerId}`, {
                method: 'PUT',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(data)
            });
        } else {
            response = await fetch('/api/bark/servers', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(data)
            });
        }
        
        const result = await response.json();
        
        if (response.ok) {
            showAlert(currentEditingServerId ? '服务器更新成功' : '服务器创建成功', 'success');
            closeServerModal();
            loadServers(currentServerPage);
        } else {
            showAlert('保存失败: ' + (result.error || '未知错误'), 'error');
        }
    } catch (error) {
        console.error('Error saving server:', error);
        showAlert('保存失败: ' + error.message, 'error');
    }
}

// 编辑服务器
function editServer(serverId) {
    showServerModal(serverId);
}

// 删除服务器
async function deleteServer(serverId, serverName) {
    Utils.showConfirm(
        '删除服务器',
        `确定要删除服务器 "${serverName}" 吗？此操作不可恢复。`,
        async function() {
            try {
                const response = await fetch(`/api/bark/servers/${serverId}`, {
                    method: 'DELETE'
                });
                
                const result = await response.json();
                
                if (response.ok) {
                    showAlert('服务器删除成功', 'success');
                    loadServers(currentServerPage);
                } else {
                    showAlert('删除失败: ' + (result.error || '未知错误'), 'error');
                }
            } catch (error) {
                console.error('Error deleting server:', error);
                showAlert('删除失败: ' + error.message, 'error');
            }
        }
    );
}

// 关闭服务器模态框
function closeServerModal() {
    document.getElementById('server-modal').classList.add('hidden');
    currentEditingServerId = null;
}

// ===== 设备管理 =====

// 加载设备列表
async function loadDevices(page = 1) {
    currentDevicePage = page;
    const loading = document.getElementById('devices-loading');
    const tableContainer = document.getElementById('devices-table-container');
    
    loading.classList.remove('hidden');
    tableContainer.classList.add('hidden');
    
    try {
        const response = await fetch(`/api/bark/devices?page=${page}&limit=10`);
        const data = await response.json();
        
        if (response.ok) {
            renderDevicesTable(data.devices || []);
            renderDevicesPagination(data.total || 0, data.page || 1, data.limit || 10);
        } else {
            showAlert('加载设备列表失败: ' + (data.error || '未知错误'), 'error');
        }
    } catch (error) {
        console.error('Error loading devices:', error);
        showAlert('加载设备列表失败: ' + error.message, 'error');
    } finally {
        loading.classList.add('hidden');
        tableContainer.classList.remove('hidden');
    }
}

// 渲染设备表格
function renderDevicesTable(devices) {
    const tbody = document.getElementById('devices-tbody');
    
    if (devices.length === 0) {
        tbody.innerHTML = `
            <tr>
                <td colspan="8" class="px-6 py-12 text-center">
                    <div class="flex flex-col items-center">
                        <i data-lucide="smartphone" class="w-12 h-12 text-slate-300 mb-4"></i>
                        <h3 class="text-sm font-medium text-slate-900 mb-1">暂无设备配置</h3>
                        <p class="text-sm text-slate-500 mb-4">开始添加您的第一个 Bark 设备</p>
                        <button onclick="showDeviceModal()" class="inline-flex items-center px-4 py-2 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-700 transition-colors">
                            <i data-lucide="plus" class="w-4 h-4 mr-2"></i>
                            添加设备
                        </button>
                    </div>
                </td>
            </tr>
        `;
        // 重新初始化图标
        lucide.createIcons();
        return;
    }
    
    tbody.innerHTML = devices.map(device => `
        <tr class="hover:bg-slate-50">
            <td class="px-6 py-4 whitespace-nowrap">
                <div class="flex items-center">
                    <div class="text-sm font-medium text-slate-900">${escapeHtml(device.name)}</div>
                    ${device.is_default ? '<span class="ml-2 inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-yellow-100 text-yellow-800">默认</span>' : ''}
                </div>
            </td>
            <td class="px-6 py-4 whitespace-nowrap">
                <code class="text-sm text-slate-600 bg-slate-100 px-2 py-1 rounded">${escapeHtml(device.device_key)}</code>
            </td>
            <td class="px-6 py-4 whitespace-nowrap">
                <div class="text-sm text-slate-600">${device.server ? escapeHtml(device.server.name) : '-'}</div>
            </td>
            <td class="px-6 py-4">
                <div class="text-sm text-slate-600 max-w-xs truncate">${escapeHtml(device.description || '-')}</div>
            </td>
            <td class="px-6 py-4 whitespace-nowrap">
                <span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${device.status === 'active' ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'}">
                    ${device.status === 'active' ? '启用' : '禁用'}
                </span>
            </td>
            <td class="px-6 py-4 whitespace-nowrap text-sm text-slate-600">
                ${device.is_default ? '是' : '否'}
            </td>
            <td class="px-6 py-4 whitespace-nowrap text-sm text-slate-600">
                ${formatDateTime(device.created_at)}
            </td>
            <td class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                <div class="flex items-center justify-end space-x-2">
                    <button onclick="editDevice(${device.id})" class="text-blue-600 hover:text-blue-900">编辑</button>
                    <button onclick="deleteDevice(${device.id}, '${escapeHtml(device.name)}')" class="text-red-600 hover:text-red-900">删除</button>
                </div>
            </td>
        </tr>
    `).join('');
}

// 渲染设备分页
function renderDevicesPagination(total, page, limit) {
    const totalPages = Math.ceil(total / limit);
    const pagination = document.getElementById('devices-pagination');
    
    if (totalPages <= 1) {
        pagination.innerHTML = '';
        return;
    }
    
    let html = `
        <button onclick="loadDevices(${page - 1})" ${page <= 1 ? 'disabled' : ''} 
                class="px-3 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-300 rounded-lg hover:bg-slate-50 disabled:opacity-50 disabled:cursor-not-allowed">
            上一页
        </button>
        <span class="px-3 py-2 text-sm text-slate-600">第 ${page} 页，共 ${totalPages} 页</span>
        <button onclick="loadDevices(${page + 1})" ${page >= totalPages ? 'disabled' : ''} 
                class="px-3 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-300 rounded-lg hover:bg-slate-50 disabled:opacity-50 disabled:cursor-not-allowed">
            下一页
        </button>
    `;
    
    pagination.innerHTML = html;
}

// 显示设备模态框
async function showDeviceModal(deviceId = null) {
    currentEditingDeviceId = deviceId;
    const modal = document.getElementById('device-modal');
    const title = document.getElementById('device-modal-title');
    const form = document.getElementById('device-form');
    
    // 重置表单
    form.reset();
    
    // 加载服务器选项
    await loadServerOptions();
    
    if (deviceId) {
        title.textContent = '编辑设备';
        loadDeviceData(deviceId);
    } else {
        title.textContent = '添加设备';
    }
    
    modal.classList.remove('hidden');
}

// 加载服务器选项
async function loadServerOptions() {
    try {
        const response = await fetch('/api/bark/servers?limit=100');
        const data = await response.json();
        
        const select = document.getElementById('device-server');
        select.innerHTML = '<option value="">选择服务器（可选）</option>';
        
        if (response.ok && data.servers) {
            data.servers.forEach(server => {
                if (server.status === 'active') {
                    const option = document.createElement('option');
                    option.value = server.id;
                    option.textContent = server.name;
                    select.appendChild(option);
                }
            });
        }
    } catch (error) {
        console.error('Error loading server options:', error);
    }
}

// 加载设备数据
async function loadDeviceData(deviceId) {
    try {
        const response = await fetch(`/api/bark/devices/${deviceId}`);
        const device = await response.json();
        
        if (response.ok) {
            document.getElementById('device-name').value = device.name || '';
            document.getElementById('device-key').value = device.device_key || '';
            document.getElementById('device-server').value = device.server_id || '';
            document.getElementById('device-description').value = device.description || '';
            document.getElementById('device-is-default').checked = device.is_default || false;
        } else {
            showAlert('加载设备数据失败: ' + (device.error || '未知错误'), 'error');
        }
    } catch (error) {
        console.error('Error loading device data:', error);
        showAlert('加载设备数据失败: ' + error.message, 'error');
    }
}

// 保存设备
async function saveDevice() {
    const name = document.getElementById('device-name').value.trim();
    const deviceKey = document.getElementById('device-key').value.trim();
    const serverId = document.getElementById('device-server').value;
    const description = document.getElementById('device-description').value.trim();
    const isDefault = document.getElementById('device-is-default').checked;
    
    if (!name || !deviceKey) {
        showAlert('请填写必填字段', 'error');
        return;
    }
    
    const data = {
        name,
        device_key: deviceKey,
        server_id: serverId ? parseInt(serverId) : 0,
        description,
        is_default: isDefault
    };
    
    try {
        let response;
        if (currentEditingDeviceId) {
            response = await fetch(`/api/bark/devices/${currentEditingDeviceId}`, {
                method: 'PUT',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(data)
            });
        } else {
            response = await fetch('/api/bark/devices', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(data)
            });
        }
        
        const result = await response.json();
        
        if (response.ok) {
            showAlert(currentEditingDeviceId ? '设备更新成功' : '设备创建成功', 'success');
            closeDeviceModal();
            loadDevices(currentDevicePage);
        } else {
            showAlert('保存失败: ' + (result.error || '未知错误'), 'error');
        }
    } catch (error) {
        console.error('Error saving device:', error);
        showAlert('保存失败: ' + error.message, 'error');
    }
}

// 编辑设备
function editDevice(deviceId) {
    showDeviceModal(deviceId);
}

// 删除设备
async function deleteDevice(deviceId, deviceName) {
    Utils.showConfirm(
        '删除设备',
        `确定要删除设备 "${deviceName}" 吗？此操作不可恢复。`,
        async function() {
            try {
                const response = await fetch(`/api/bark/devices/${deviceId}`, {
                    method: 'DELETE'
                });
                
                const result = await response.json();
                
                if (response.ok) {
                    showAlert('设备删除成功', 'success');
                    loadDevices(currentDevicePage);
                } else {
                    showAlert('删除失败: ' + (result.error || '未知错误'), 'error');
                }
            } catch (error) {
                console.error('Error deleting device:', error);
                showAlert('删除失败: ' + error.message, 'error');
            }
        }
    );
}

// 关闭设备模态框
function closeDeviceModal() {
    document.getElementById('device-modal').classList.add('hidden');
    currentEditingDeviceId = null;
}

// ===== 工具函数 =====

// 显示提示消息
function showAlert(message, type = 'success') {
    const container = document.getElementById('alert-container');
    const alertDiv = document.createElement('div');
    
    const bgColor = type === 'success' ? 'bg-green-50 border-green-200 text-green-800' : 'bg-red-50 border-red-200 text-red-800';
    const iconName = type === 'success' ? 'check-circle' : 'alert-circle';
    
    alertDiv.className = `flex items-center p-4 border rounded-lg shadow-sm ${bgColor}`;
    alertDiv.innerHTML = `
        <i data-lucide="${iconName}" class="w-5 h-5 mr-3"></i>
        <span class="text-sm font-medium">${escapeHtml(message)}</span>
        <button onclick="this.parentElement.remove()" class="ml-auto text-current hover:opacity-70">
            <i data-lucide="x" class="w-4 h-4"></i>
        </button>
    `;
    
    // 添加到容器
    container.appendChild(alertDiv);
    
    // 初始化图标
    lucide.createIcons();
    
    // 3秒后自动移除
    setTimeout(() => {
        if (alertDiv.parentNode) {
            alertDiv.parentNode.removeChild(alertDiv);
        }
    }, 3000);
}

// HTML转义
function escapeHtml(text) {
    if (!text) return '';
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// 格式化日期时间
function formatDateTime(dateString) {
    if (!dateString) return '-';
    const date = new Date(dateString);
    return date.toLocaleString('zh-CN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit'
    });
}