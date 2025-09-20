// 日志页面 JavaScript

let currentPage = 1;
let currentFilters = {
    task_id: '',
    status: '',
    date: ''
};
let isLoading = false;
let allTasks = [];

// 页面加载完成后初始化
$(document).ready(function() {
    initializeLogsPage();
    loadTasks();
    loadLogs();
    bindEvents();
});

// 初始化日志页面
function initializeLogsPage() {
    // 从URL参数获取初始筛选条件
    const urlParams = new URLSearchParams(window.location.search);
    if (urlParams.get('task_id')) {
        currentFilters.task_id = urlParams.get('task_id');
    }
    
    console.log('Logs page initialized', { currentFilters });
}

// 绑定事件
function bindEvents() {
    // 搜索按钮
    $('#searchBtn').on('click', function() {
        applyFilters();
    });
    
    // 刷新按钮
    $('#refreshBtn').on('click', function() {
        loadLogs();
    });
    
    // 删除所有日志按钮
    $('#deleteAllLogsBtn').on('click', function() {
        deleteAllLogs();
    });
    
    // 删除所有Bark日志按钮
    $('#deleteAllBarkLogsBtn').on('click', function() {
        deleteAllBarkLogs();
    });
    
    // 统计信息按钮
    $('#logStatsBtn').on('click', function() {
        showLogStats();
    });
    
    // 筛选器变化
    $('#taskFilter, #statusFilter, #dateFilter').on('change', function() {
        applyFilters();
    });
    
    // Enter键搜索
    $('#dateFilter').on('keypress', function(e) {
        if (e.which === 13) {
            applyFilters();
        }
    });
}

// 加载任务列表（用于筛选器）
async function loadTasks() {
    try {
        const response = await Utils.api.get('/api/tasks?limit=1000'); // 获取所有任务
        allTasks = response.tasks || [];
        
        // 填充任务筛选器
        const taskFilter = $('#taskFilter');
        taskFilter.empty().append('<option value="">所有任务</option>');
        
        allTasks.forEach(task => {
            const selected = task.id.toString() === currentFilters.task_id ? 'selected' : '';
            taskFilter.append(`<option value="${task.id}" ${selected}>${task.name}</option>`);
        });
        
    } catch (error) {
        console.error('Failed to load tasks:', error);
    }
}

// 应用筛选条件
function applyFilters() {
    currentFilters.task_id = $('#taskFilter').val();
    currentFilters.status = $('#statusFilter').val();
    currentFilters.date = $('#dateFilter').val();
    currentPage = 1;
    
    loadLogs();
}

// 加载日志列表
async function loadLogs() {
    if (isLoading) return;
    
    isLoading = true;
    const container = $('#logsContainer');
    
    try {
        // 显示加载状态
        showLoadingState(container);
        
        // 构建查询参数
        const params = new URLSearchParams({
            page: currentPage,
            limit: APP_CONFIG.PAGE_SIZE
        });
        
        // 添加筛选条件
        Object.keys(currentFilters).forEach(key => {
            if (currentFilters[key]) {
                params.append(key, currentFilters[key]);
            }
        });
        
        // 请求数据 - 如果有特定任务ID，请求该任务的日志
        let url = '/tasks/logs';
        if (currentFilters.task_id) {
            url = `/tasks/${currentFilters.task_id}/logs`;
        }
        
        // 由于后端API结构，我们需要获取所有日志并在前端筛选
        const response = await getAllLogs(params);
        
        // 保存当前日志数据供详情查看使用
        window.currentLogs = response.logs;
        
        // 渲染日志列表
        renderLogs(response.logs);
        renderPagination(response.total, response.page, response.limit);
        
    } catch (error) {
        console.error('Failed to load logs:', error);
        showErrorState(container, '加载日志失败: ' + error.message);
    } finally {
        isLoading = false;
    }
}

// 获取所有日志（临时方案，实际应该在后端实现全局日志API）
async function getAllLogs(params) {
    if (currentFilters.task_id) {
        // 获取特定任务的日志
        const response = await Utils.api.get(`/api/tasks/${currentFilters.task_id}/logs?${params}`);
        // 保存日志数据
        window.currentLogs = response.logs;
        return response;
    } else {
        // 获取所有任务的日志（需要遍历所有任务）
        let allLogs = [];
        
        for (const task of allTasks) {
            try {
                const taskLogs = await Utils.api.get(`/api/tasks/${task.id}/logs?limit=1000`);
                if (taskLogs.logs) {
                    allLogs = allLogs.concat(taskLogs.logs);
                }
            } catch (error) {
                console.error(`Failed to load logs for task ${task.id}:`, error);
            }
        }
        
        // 按状态筛选
        if (currentFilters.status) {
            allLogs = allLogs.filter(log => log.status === currentFilters.status);
        }
        
        // 按日期筛选
        if (currentFilters.date) {
            const filterDate = new Date(currentFilters.date);
            allLogs = allLogs.filter(log => {
                const logDate = new Date(log.start_time);
                return logDate.toDateString() === filterDate.toDateString();
            });
        }
        
        // 排序
        allLogs.sort((a, b) => new Date(b.start_time) - new Date(a.start_time));
        
        // 分页
        const page = parseInt(params.get('page')) || 1;
        const limit = parseInt(params.get('limit')) || APP_CONFIG.PAGE_SIZE;
        const total = allLogs.length;
        const offset = (page - 1) * limit;
        const logs = allLogs.slice(offset, offset + limit);
        
        // 保存日志数据
        window.currentLogs = logs;
        
        return { logs, total, page, limit };
    }
}

// 显示加载状态
function showLoadingState(container) {
    container.html(`
        <div class="flex items-center justify-center py-12">
            <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
            <span class="ml-3 text-slate-600">正在加载日志...</span>
        </div>
    `);
}

// 显示错误状态
function showErrorState(container, message) {
    container.html(`
        <div class="flex flex-col items-center justify-center py-12 text-center">
            <div class="w-12 h-12 bg-red-100 rounded-full flex items-center justify-center mb-4">
                <i data-lucide="alert-triangle" class="w-6 h-6 text-red-600"></i>
            </div>
            <h3 class="text-lg font-semibold text-slate-900 mb-2">加载失败</h3>
            <p class="text-slate-600 mb-4">${message}</p>
            <button onclick="loadLogs()" class="inline-flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-xl hover:bg-blue-700 transition-colors">
                <i data-lucide="refresh-cw" class="w-4 h-4"></i>
                重试
            </button>
        </div>
    `);
    lucide.createIcons();
}

// 渲染日志列表
function renderLogs(logs) {
    const container = $('#logsContainer');
    
    if (!logs || logs.length === 0) {
        container.html(`
            <div class="flex flex-col items-center justify-center py-12 text-center">
                <div class="w-16 h-16 bg-slate-100 rounded-full flex items-center justify-center mb-4">
                    <i data-lucide="file-text" class="w-8 h-8 text-slate-400"></i>
                </div>
                <h3 class="text-lg font-semibold text-slate-900 mb-2">暂无日志</h3>
                <p class="text-slate-600">还没有任务执行日志</p>
            </div>
        `);
        lucide.createIcons();
        return;
    }
    
    let html = '<div class="space-y-0">';
    
    logs.forEach(log => {
        html += renderLogRow(log);
    });
    
    html += '</div>';
    container.html(html);
    lucide.createIcons();
}

// 渲染日志行
function renderLogRow(log) {
    const statusBadge = Utils.getStatusBadge(log.status);
    const startTime = Utils.formatDateTime(log.start_time);
    const duration = Utils.formatDuration(log.duration);
    const taskName = log.task ? log.task.name : `任务 ${log.task_id}`;
    
    return `
        <div class="bg-white border-b border-slate-200 p-3 hover:bg-slate-50 transition-colors">
            <div class="flex items-center justify-between">
                <!-- 任务名称 -->
                <div class="min-w-0 flex-1 mr-4">
                    <div class="text-sm font-medium text-slate-900 truncate" title="${taskName}">
                        ${taskName}
                    </div>
                    <div class="text-xs text-slate-500">ID: ${log.task_id}</div>
                </div>
                
                <!-- 状态 -->
                <div class="flex-shrink-0 mr-4">
                    ${statusBadge}
                </div>
                
                <!-- 开始时间 -->
                <div class="hidden sm:block min-w-0 flex-shrink-0 mr-4">
                    <div class="text-sm text-slate-600">
                        <i data-lucide="clock" class="w-4 h-4 inline mr-1"></i>
                        <span class="truncate" title="${startTime}">${startTime}</span>
                    </div>
                </div>
                
                <!-- 执行时长 -->
                <div class="hidden md:block min-w-0 flex-shrink-0 mr-4">
                    <div class="text-sm text-slate-600">
                        <i data-lucide="timer" class="w-4 h-4 inline mr-1"></i>
                        <span>${duration}</span>
                    </div>
                </div>
                
                <!-- 相对时间 -->
                <div class="hidden lg:block min-w-0 flex-shrink-0 mr-4">
                    <div class="text-xs text-slate-500">
                        ${Utils.formatRelativeTime(log.start_time)}
                    </div>
                </div>
                
                <!-- 操作按钮 -->
                <div class="flex-shrink-0">
                    <button onclick="viewLogDetails(${log.id})" 
                            class="inline-flex items-center gap-1 px-2 py-1 bg-blue-50 text-blue-700 rounded hover:bg-blue-100 transition-colors text-sm">
                        <i data-lucide="eye" class="w-4 h-4"></i>
                        <span class="hidden sm:inline">详情</span>
                    </button>
                </div>
            </div>
        </div>
    `;
}

// 渲染分页
function renderPagination(total, page, limit) {
    const totalPages = Math.ceil(total / limit);
    const pagination = $('#pagination');
    const paginationList = $('#paginationList');
    
    if (totalPages <= 1) {
        pagination.addClass('hidden');
        return;
    }
    
    let html = '';
    
    // 上一页
    const prevDisabled = page <= 1;
    html += `
        <button onclick="changePage(${page - 1})" 
                ${prevDisabled ? 'disabled' : ''} 
                class="px-3 py-2 text-sm font-medium text-slate-500 bg-white border border-slate-300 rounded-l-lg hover:bg-slate-50 hover:text-slate-700 ${prevDisabled ? 'cursor-not-allowed opacity-50' : ''}">
            <i data-lucide="chevron-left" class="w-4 h-4"></i>
        </button>
    `;
    
    // 页码
    const startPage = Math.max(1, page - 2);
    const endPage = Math.min(totalPages, page + 2);
    
    if (startPage > 1) {
        html += `<button onclick="changePage(1)" class="px-3 py-2 text-sm font-medium text-slate-500 bg-white border-t border-b border-slate-300 hover:bg-slate-50 hover:text-slate-700">1</button>`;
        if (startPage > 2) {
            html += `<span class="px-3 py-2 text-sm font-medium text-slate-500 bg-white border-t border-b border-slate-300">...</span>`;
        }
    }
    
    for (let i = startPage; i <= endPage; i++) {
        const active = i === page;
        html += `
            <button onclick="changePage(${i})" 
                    class="px-3 py-2 text-sm font-medium ${active ? 'text-blue-600 bg-blue-50 border-blue-500' : 'text-slate-500 bg-white hover:bg-slate-50 hover:text-slate-700'} border-t border-b border-slate-300">
                ${i}
            </button>
        `;
    }
    
    if (endPage < totalPages) {
        if (endPage < totalPages - 1) {
            html += `<span class="px-3 py-2 text-sm font-medium text-slate-500 bg-white border-t border-b border-slate-300">...</span>`;
        }
        html += `<button onclick="changePage(${totalPages})" class="px-3 py-2 text-sm font-medium text-slate-500 bg-white border-t border-b border-slate-300 hover:bg-slate-50 hover:text-slate-700">${totalPages}</button>`;
    }
    
    // 下一页
    const nextDisabled = page >= totalPages;
    html += `
        <button onclick="changePage(${page + 1})" 
                ${nextDisabled ? 'disabled' : ''} 
                class="px-3 py-2 text-sm font-medium text-slate-500 bg-white border border-slate-300 rounded-r-lg hover:bg-slate-50 hover:text-slate-700 ${nextDisabled ? 'cursor-not-allowed opacity-50' : ''}">
            <i data-lucide="chevron-right" class="w-4 h-4"></i>
        </button>
    `;
    
    paginationList.html(html);
    pagination.removeClass('hidden');
    lucide.createIcons();
}

// 切换页面
function changePage(page) {
    if (page < 1 || page === currentPage) return;
    
    currentPage = page;
    loadLogs();
}

// 查看日志详情
async function viewLogDetails(logId) {
    try {
        // 从当前加载的日志中查找
        const log = await findLogById(logId);
        
        if (!log) {
            Utils.showToast('日志不存在', 'error');
            return;
        }
        
        showLogDetailsModal(log);
        
    } catch (error) {
        console.error('Failed to load log details:', error);
        Utils.showToast('加载日志详情失败: ' + error.message, 'error');
    }
}

// 查找日志详情
async function findLogById(logId) {
    // 首先尝试从当前页面的日志数据中查找
    if (window.currentLogs) {
        const log = window.currentLogs.find(l => l.id === logId);
        if (log) {
            return log;
        }
    }
    
    // 如果当前页面没有，尝试通过API获取
    // 由于没有单独的日志详情API，我们需要通过任务日志API获取
    try {
        // 这里需要知道任务ID，暂时从URL参数获取
        const urlParams = new URLSearchParams(window.location.search);
        const taskId = urlParams.get('task_id') || currentFilters.task_id;
        
        if (taskId) {
            const response = await Utils.api.get(`/api/tasks/${taskId}/logs?limit=100`);
            const log = response.logs.find(l => l.id === logId);
            if (log) {
                return log;
            }
        }
        
        // 如果还是找不到，返回错误
        throw new Error('日志不存在');
        
    } catch (error) {
        console.error('Failed to fetch log details:', error);
        throw error;
    }
}

// 显示日志详情模态框
function showLogDetailsModal(log) {
    const modal = document.getElementById('logDetailModal');
    const detailsContainer = document.getElementById('logDetailContent');
    
    const statusBadge = Utils.getStatusBadge(log.status);
    const startTime = Utils.formatDateTime(log.start_time);
    const endTime = Utils.formatDateTime(log.end_time);
    const duration = Utils.formatDuration(log.duration);
    
    const html = `
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
            <div class="bg-slate-50 rounded-lg p-4">
                <div class="text-sm font-medium text-slate-700 mb-1">任务名称</div>
                <div class="text-slate-900">${log.task ? log.task.name : `任务 ${log.task_id}`}</div>
            </div>
            <div class="bg-slate-50 rounded-lg p-4">
                <div class="text-sm font-medium text-slate-700 mb-1">状态</div>
                <div>${statusBadge}</div>
            </div>
            <div class="bg-slate-50 rounded-lg p-4">
                <div class="text-sm font-medium text-slate-700 mb-1">开始时间</div>
                <div class="text-slate-900">${startTime}</div>
            </div>
            <div class="bg-slate-50 rounded-lg p-4">
                <div class="text-sm font-medium text-slate-700 mb-1">结束时间</div>
                <div class="text-slate-900">${endTime}</div>
            </div>
            <div class="bg-slate-50 rounded-lg p-4">
                <div class="text-sm font-medium text-slate-700 mb-1">执行时长</div>
                <div class="text-slate-900">${duration}</div>
            </div>
            <div class="bg-slate-50 rounded-lg p-4">
                <div class="text-sm font-medium text-slate-700 mb-1">日志ID</div>
                <div class="text-slate-900">${log.id}</div>
            </div>
        </div>
        
        ${log.output ? `
            <div class="mb-6">
                <div class="text-sm font-medium text-slate-700 mb-2 flex items-center">
                    <i data-lucide="terminal" class="w-4 h-4 mr-2 text-green-600"></i>
                    标准输出
                </div>
                <div class="bg-slate-900 text-green-400 p-4 rounded-lg font-mono text-sm overflow-auto max-h-80 border border-slate-700 log-output-scrollable log-output-dark">
                    <pre class="whitespace-pre-wrap break-words">${escapeHtmlWithNewlines(log.output)}</pre>
                </div>
            </div>
        ` : ''}
        
        ${log.error ? `
            <div class="mb-6">
                <div class="text-sm font-medium text-slate-700 mb-2 flex items-center">
                    <i data-lucide="alert-triangle" class="w-4 h-4 mr-2 text-red-600"></i>
                    错误输出
                </div>
                <div class="bg-red-50 border border-red-200 text-red-800 p-4 rounded-lg font-mono text-sm overflow-auto max-h-80 log-output-scrollable">
                    <pre class="whitespace-pre-wrap break-words">${escapeHtmlWithNewlines(log.error)}</pre>
                </div>
            </div>
        ` : ''}
    `;
    
    detailsContainer.innerHTML = html;
    modal.classList.remove('hidden');
    
    // 添加ESC键监听器
    document.addEventListener('keydown', handleEscapeKey);
}

// HTML转义
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// HTML转义并处理换行符
function escapeHtmlWithNewlines(text) {
    if (!text) return '';
    
    // 先进行HTML转义
    const div = document.createElement('div');
    div.textContent = text;
    let escaped = div.innerHTML;
    
    // 将 \n 替换为真正的换行符
    escaped = escaped.replace(/\\n/g, '\n');
    
    return escaped;
}

// 删除所有日志
async function deleteAllLogs() {
    Utils.showConfirm(
        '删除所有日志',
        '确定要删除所有执行日志吗？此操作无法撤销，将清空所有任务的执行历史记录。',
        async function() {
            try {
                const response = await Utils.api.delete('/api/logs/all');
                Utils.showToast(`所有日志已删除 - 共删除 ${response.deleted_count} 条记录`, 'success');
                
                // 刷新日志列表
                setTimeout(() => {
                    loadLogs();
                }, 1000);
                
            } catch (error) {
                console.error('Failed to delete all logs:', error);
                Utils.showToast('删除日志失败: ' + error.message, 'error');
            }
        }
    );
}

// 删除所有Bark日志
async function deleteAllBarkLogs() {
    Utils.showConfirm(
        '删除所有Bark日志',
        '确定要删除所有Bark通知记录吗？此操作无法撤销，将清空所有Bark发送历史记录。',
        async function() {
            try {
                const response = await Utils.api.delete('/api/bark/records/all');
                Utils.showToast(`所有Bark日志已删除 - 共删除 ${response.deleted_count} 条记录`, 'success');
            } catch (error) {
                console.error('Failed to delete all bark logs:', error);
                Utils.showToast('删除Bark日志失败: ' + error.message, 'error');
            }
        }
    );
}

// 显示日志统计信息
async function showLogStats() {
    const modal = document.getElementById('statsModal');
    modal.classList.remove('hidden');
    const statsContent = $('#statsContent');
    
    try {
        // 显示加载状态
        statsContent.html(`
            <div class="text-center">
                <div class="spinner-border" role="status">
                    <span class="visually-hidden">加载中...</span>
                </div>
                <p class="mt-2">正在加载统计信息...</p>
            </div>
        `);
        
        // 并行获取日志统计和Bark统计
        const [logStats, barkStats] = await Promise.all([
            Utils.api.get('/api/logs/stats'),
            Utils.api.get('/api/bark/stats')
        ]);
        
        let html = `
            <!-- 任务日志统计 -->
            <div class="mb-4">
                <h3 class="text-base font-semibold text-slate-800 mb-3 flex items-center">
                    <i data-lucide="file-text" class="w-4 h-4 mr-2 text-slate-600"></i>
                    任务日志统计
                </h3>
                <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
                    <div class="bg-gradient-to-r from-blue-50 to-blue-100 rounded-lg p-4 text-center">
                        <div class="text-2xl font-bold text-blue-600 mb-1">${logStats.total_logs}</div>
                        <div class="text-sm text-slate-600">总日志数</div>
                        <div class="text-xs text-slate-500 mt-1">最多 ${logStats.max_total_logs} 条</div>
                    </div>
                    <div class="bg-gradient-to-r from-green-50 to-green-100 rounded-lg p-4 text-center">
                        <div class="text-2xl font-bold text-green-600 mb-1">${logStats.max_logs_per_task}</div>
                        <div class="text-sm text-slate-600">单任务限制</div>
                        <div class="text-xs text-slate-500 mt-1">每个任务最多条数</div>
                    </div>
                    <div class="bg-gradient-to-r from-purple-50 to-purple-100 rounded-lg p-4 text-center">
                        <div class="text-2xl font-bold text-purple-600 mb-1">${Math.round((logStats.total_logs / logStats.max_total_logs) * 100)}%</div>
                        <div class="text-sm text-slate-600">存储使用率</div>
                        <div class="text-xs text-slate-500 mt-1">任务日志空间占用</div>
                    </div>
                </div>
            </div>
            
            <!-- Bark通知统计 -->
            <div class="mb-4">
                <h3 class="text-base font-semibold text-slate-800 mb-3 flex items-center">
                    <i data-lucide="bell" class="w-4 h-4 mr-2 text-orange-600"></i>
                    Bark通知统计
                </h3>
                <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
                    <div class="bg-gradient-to-r from-orange-50 to-orange-100 rounded-lg p-4 text-center">
                        <div class="text-2xl font-bold text-orange-600 mb-1">${barkStats.total_records}</div>
                        <div class="text-sm text-slate-600">总通知数</div>
                        <div class="text-xs text-slate-500 mt-1">最多 ${barkStats.max_records} 条</div>
                    </div>
                    <div class="bg-gradient-to-r from-emerald-50 to-emerald-100 rounded-lg p-4 text-center">
                        <div class="text-2xl font-bold text-emerald-600 mb-1">${barkStats.success_records}</div>
                        <div class="text-sm text-slate-600">发送成功</div>
                        <div class="text-xs text-slate-500 mt-1">成功发送的通知</div>
                    </div>
                    <div class="bg-gradient-to-r from-yellow-50 to-yellow-100 rounded-lg p-4 text-center">
                        <div class="text-2xl font-bold text-yellow-600 mb-1">${barkStats.total_records - barkStats.success_records}</div>
                        <div class="text-sm text-slate-600">跳过/失败</div>
                        <div class="text-xs text-slate-500 mt-1">跳过或失败的通知</div>
                    </div>
                    <div class="bg-gradient-to-r from-indigo-50 to-indigo-100 rounded-lg p-4 text-center">
                        <div class="text-2xl font-bold text-indigo-600 mb-1">${Math.round((barkStats.total_records / barkStats.max_records) * 100)}%</div>
                        <div class="text-sm text-slate-600">存储使用率</div>
                        <div class="text-xs text-slate-500 mt-1">通知记录空间占用</div>
                    </div>
                </div>
            </div>
        `;
        
        // 时间范围信息
        if (logStats.total_logs > 0 || barkStats.total_records > 0) {
            html += `
                <div class="bg-slate-50 rounded-lg p-3 mb-4">
                    <h4 class="font-semibold text-slate-900 mb-2 text-sm">时间范围</h4>
                    <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
            `;
            
            if (logStats.total_logs > 0) {
                html += `
                        <div class="text-xs">
                            <div class="font-medium text-slate-700 mb-1">任务日志</div>
                            <div class="text-slate-600">${Utils.formatDateTime(logStats.oldest_log)} 至 ${Utils.formatDateTime(logStats.newest_log)}</div>
                        </div>
                `;
            }
            
            if (barkStats.total_records > 0) {
                html += `
                        <div class="text-xs">
                            <div class="font-medium text-slate-700 mb-1">通知记录</div>
                            <div class="text-slate-600">${Utils.formatDateTime(barkStats.oldest_record)} 至 ${Utils.formatDateTime(barkStats.newest_record)}</div>
                        </div>
                `;
            }
            
            html += `
                    </div>
                </div>
            `;
        }
        
        // 添加智能管理说明
        html += `
            <div class="bg-gradient-to-r from-slate-50 to-slate-100 rounded-lg p-4">
                <h3 class="text-base font-semibold text-slate-800 mb-3 flex items-center">
                    <i data-lucide="brain" class="w-4 h-4 mr-2 text-slate-600"></i>
                    智能存储管理
                </h3>
                <div class="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm text-slate-600">
                    <div class="flex items-start">
                        <i data-lucide="database" class="w-4 h-4 mr-2 mt-0.5 text-slate-500 flex-shrink-0"></i>
                        <div>
                            <div class="font-medium mb-1">任务日志</div>
                            <div class="text-xs">每个任务最多保留 ${logStats.max_logs_per_task} 条，全局最多 ${logStats.max_total_logs} 条</div>
                        </div>
                    </div>
                    <div class="flex items-start">
                        <i data-lucide="bell" class="w-4 h-4 mr-2 mt-0.5 text-orange-500 flex-shrink-0"></i>
                        <div>
                            <div class="font-medium mb-1">通知记录</div>
                            <div class="text-xs">Bark通知记录最多保留 ${barkStats.max_records} 条，包含去重检查</div>
                        </div>
                    </div>
                    <div class="flex items-start">
                        <i data-lucide="trash-2" class="w-4 h-4 mr-2 mt-0.5 text-slate-500 flex-shrink-0"></i>
                        <div>
                            <div class="font-medium mb-1">自动清理</div>
                            <div class="text-xs">超过限制时自动删除最旧的记录，保持系统性能</div>
                        </div>
                    </div>
                    <div class="flex items-start">
                        <i data-lucide="filter" class="w-4 h-4 mr-2 mt-0.5 text-purple-500 flex-shrink-0"></i>
                        <div>
                            <div class="font-medium mb-1">去重机制</div>
                            <div class="text-xs">通知去重仅基于成功发送的记录，跳过记录不占用配额</div>
                        </div>
                    </div>
                </div>
            </div>
        `;
        
        statsContent.html(html);
        lucide.createIcons();
        
        // 添加ESC键监听器
        document.addEventListener('keydown', handleEscapeKey);
        
    } catch (error) {
        console.error('Failed to load log stats:', error);
        statsContent.html(`
            <div class="bg-red-50 border border-red-200 rounded-xl p-4">
                <div class="flex items-center">
                    <i data-lucide="alert-triangle" class="w-5 h-5 text-red-600 mr-2"></i>
                    <div class="text-sm text-red-800">
                        加载统计信息失败: ${error.message}
                    </div>
                </div>
            </div>
        `);
        lucide.createIcons();
    }
}

// 关闭日志详情模态框
function closeLogDetailModal() {
    const modal = document.getElementById('logDetailModal');
    modal.classList.add('hidden');
    // 移除ESC键监听器
    document.removeEventListener('keydown', handleEscapeKey);
}

// 关闭统计模态框
function closeStatsModal() {
    const modal = document.getElementById('statsModal');
    modal.classList.add('hidden');
    // 移除ESC键监听器
    document.removeEventListener('keydown', handleEscapeKey);
}

// 处理ESC键关闭模态框
function handleEscapeKey(event) {
    if (event.key === 'Escape') {
        const logDetailModal = document.getElementById('logDetailModal');
        const statsModal = document.getElementById('statsModal');
        
        if (!logDetailModal.classList.contains('hidden')) {
            closeLogDetailModal();
        } else if (!statsModal.classList.contains('hidden')) {
            closeStatsModal();
        }
    }
}

// 导出函数供HTML调用
window.changePage = changePage;
window.viewLogDetails = viewLogDetails;
window.closeLogDetailModal = closeLogDetailModal;
window.closeStatsModal = closeStatsModal;
