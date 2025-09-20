// 任务管理页面 JavaScript

let currentPage = 1;
let currentStatus = '';
let isLoading = false;

// 页面加载完成后初始化
$(document).ready(function() {
    initializeTasksPage();
    loadTasks();
    
    // 绑定事件
    $('#statusFilter').on('change', function() {
        currentStatus = $(this).val();
        currentPage = 1;
        loadTasks();
    });
    
    $('#refreshBtn').on('click', function() {
        loadTasks();
    });
});

// 初始化任务页面
function initializeTasksPage() {
    console.log('Tasks page initialized');
}

// 加载任务列表
async function loadTasks() {
    if (isLoading) return;
    
    isLoading = true;
    const container = $('#tasksContainer');
    
    try {
        // 显示加载状态
        showLoadingState(container);
        
        // 构建查询参数
        const params = new URLSearchParams({
            page: currentPage,
            limit: APP_CONFIG.PAGE_SIZE
        });
        
        if (currentStatus) {
            params.append('status', currentStatus);
        }
        
        // 请求数据
        const response = await Utils.api.get(`/api/tasks?${params}`);
        
        // 渲染任务列表
        renderTasks(response.tasks);
        renderPagination(response.total, response.page, response.limit);
        
    } catch (error) {
        console.error('Failed to load tasks:', error);
        showErrorState(container, '加载任务列表失败: ' + error.message);
    } finally {
        isLoading = false;
    }
}

// 显示加载状态
function showLoadingState(container) {
    container.html(`
        <div class="flex items-center justify-center py-12">
            <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
            <span class="ml-3 text-slate-600">正在加载任务列表...</span>
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
            <button onclick="loadTasks()" class="inline-flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-xl hover:bg-blue-700 transition-colors">
                <i data-lucide="refresh-cw" class="w-4 h-4"></i>
                重试
            </button>
        </div>
    `);
    lucide.createIcons();
}

// 渲染任务列表
function renderTasks(tasks) {
    const container = $('#tasksContainer');
    
    if (!tasks || tasks.length === 0) {
        container.html(`
            <div class="flex flex-col items-center justify-center py-12 text-center">
                <div class="w-16 h-16 bg-slate-100 rounded-full flex items-center justify-center mb-4">
                    <i data-lucide="inbox" class="w-8 h-8 text-slate-400"></i>
                </div>
                <h3 class="text-lg font-semibold text-slate-900 mb-2">暂无任务</h3>
                <p class="text-slate-600 mb-6">还没有创建任何任务，点击按钮创建第一个任务吧！</p>
                <a href="/tasks/new" class="inline-flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-xl hover:bg-blue-700 transition-colors">
                    <i data-lucide="plus" class="w-4 h-4"></i>
                    创建任务
                </a>
            </div>
        `);
        lucide.createIcons();
        return;
    }
    
    let html = '<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">';
    
    tasks.forEach(task => {
        html += renderTaskCard(task);
    });
    
    html += '</div>';
    container.html(html);
    lucide.createIcons();
    
    // 绑定状态切换事件
    bindStatusToggleEvents();
}

// 渲染单个任务卡片
function renderTaskCard(task) {
    const statusBadge = getStatusBadge(task.status, task.id);
    const lastRun = task.last_run ? Utils.formatRelativeTime(task.last_run) : '从未执行';
    const nextRun = task.next_run ? Utils.formatDateTime(task.next_run) : '-';
    const taskNameEscaped = task.name.replace(/'/g, "\\'");
    
    return `
        <div class="task-card bg-white rounded-xl shadow-md border border-slate-200 p-6 hover:shadow-lg transition-all duration-200">
            <!-- Header -->
            <div class="flex items-start justify-between mb-4">
                <div class="flex-1 min-w-0">
                    <h3 class="text-lg font-semibold text-slate-900 truncate" title="${task.name}">
                        ${task.name}
                    </h3>
                    <p class="text-sm text-slate-600 mt-1 line-clamp-2" title="${task.description}">
                        ${task.description || '无描述'}
                    </p>
                </div>
                ${statusBadge}
            </div>
            
            <!-- Stats -->
            <div class="space-y-3 mb-6">
                <div class="flex items-center text-sm text-slate-600">
                    <i data-lucide="clock" class="w-4 h-4 mr-2"></i>
                    <span class="font-mono text-xs bg-slate-100 px-2 py-1 rounded">${task.cron_expr}</span>
                </div>
                <div class="flex items-center text-sm text-slate-600">
                    <i data-lucide="play-circle" class="w-4 h-4 mr-2"></i>
                    <span>最后执行: ${lastRun}</span>
                </div>
                <div class="flex items-center text-sm text-slate-600">
                    <i data-lucide="calendar" class="w-4 h-4 mr-2"></i>
                    <span>下次执行: ${nextRun}</span>
                </div>
            </div>
            
            <!-- Actions -->
            <div class="flex justify-between items-center">
                <!-- 左侧按钮组 -->
                <div class="flex gap-2">
                    <a href="/tasks/${task.id}" 
                       class="flex items-center gap-1 px-3 py-1.5 bg-indigo-50 text-indigo-700 rounded-lg hover:bg-indigo-100 transition-colors text-sm font-medium">
                        <i data-lucide="eye" class="w-3.5 h-3.5"></i>
                        详情
                    </a>
                    <button onclick="runTaskNow(${task.id})" 
                            class="flex items-center gap-1 px-3 py-1.5 bg-green-50 text-green-700 rounded-lg hover:bg-green-100 transition-colors text-sm font-medium">
                        <i data-lucide="play" class="w-3.5 h-3.5"></i>
                        立即执行
                    </button>
                </div>
                
                <!-- 右侧删除按钮 -->
                <button onclick="deleteTask(${task.id}, '${taskNameEscaped}')" 
                        class="flex items-center gap-1 px-3 py-1.5 bg-red-50 text-red-700 rounded-lg hover:bg-red-100 transition-colors text-sm font-medium">
                    <i data-lucide="trash-2" class="w-3.5 h-3.5"></i>
                </button>
            </div>
        </div>
    `;
}

// 获取状态开关按钮
function getStatusBadge(status, taskId) {
    const isActive = status === 'active';
    const switchBg = isActive ? 'bg-blue-800' : 'bg-gray-300';
    const switchTransform = isActive ? 'translate-x-4' : 'translate-x-0';
    const tooltip = isActive ? '点击停用任务' : '点击激活任务';
    
    return `
        <div class="flex items-center space-x-2">
            <span class="text-xs font-medium text-slate-600">${isActive ? '已激活' : '未激活'}</span>
            <button type="button" 
                    class="status-toggle-btn relative inline-flex h-5 w-9 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 ${switchBg}" 
                    data-task-id="${taskId}" 
                    data-current-status="${status}"
                    title="${tooltip}">
                <span class="inline-block h-3 w-3 transform rounded-full bg-white transition-transform ${switchTransform}"></span>
            </button>
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
    loadTasks();
}

// 立即执行任务
async function runTaskNow(taskId) {
    try {
        await Utils.api.post(`/api/tasks/${taskId}/run`);
        Utils.showToast('任务已开始执行', 'success');
        
        // 延迟刷新，让用户看到状态变化
        setTimeout(() => {
            loadTasks();
        }, 1000);
        
    } catch (error) {
        console.error('Failed to run task:', error);
        Utils.showToast('执行任务失败: ' + error.message, 'error');
    }
}

// 查看任务日志
function viewTaskLogs(taskId) {
    window.location.href = `/logs?task_id=${taskId}`;
}

// 删除任务
function deleteTask(taskId, taskName) {
    showConfirmModal(
        '确认删除任务',
        `<div class="space-y-3">
            <p>确定要删除任务 <strong>"${taskName}"</strong> 吗？</p>
            <div class="bg-amber-50 border border-amber-200 rounded-lg p-3">
                <div class="flex items-start">
                    <i data-lucide="alert-triangle" class="w-5 h-5 text-amber-600 mt-0.5 mr-2 flex-shrink-0"></i>
                    <div class="text-sm text-amber-800">
                        <strong>注意：</strong>此操作将同时删除该任务的所有执行日志，无法撤销。
                    </div>
                </div>
            </div>
        </div>`,
        async function() {
            try {
                const response = await Utils.api.delete(`/api/tasks/${taskId}`);
                
                // 构建成功消息
                let message = `任务 "${response.task_name}" 删除成功`;
                if (response.deleted_logs > 0) {
                    message += `，同时删除了 ${response.deleted_logs} 条相关日志`;
                }
                
                Utils.showToast(message, 'success');
                loadTasks();
            } catch (error) {
                console.error('Failed to delete task:', error);
                Utils.showToast('删除任务失败: ' + error.message, 'error');
            }
        }
    );
}

// 显示确认模态框
function showConfirmModal(title, content, onConfirm) {
    const modal = document.getElementById('confirmModal');
    const modalTitle = document.getElementById('modalTitle');
    const modalBody = document.getElementById('modalBody');
    const modalConfirm = document.getElementById('modalConfirm');
    const modalCancel = document.getElementById('modalCancel');
    
    modalTitle.textContent = title;
    modalBody.innerHTML = content;
    
    // 重新初始化图标
    lucide.createIcons();
    
    // 显示模态框
    modal.classList.remove('hidden');
    
    // 绑定事件
    const confirmHandler = () => {
        modal.classList.add('hidden');
        onConfirm();
        modalConfirm.removeEventListener('click', confirmHandler);
        modalCancel.removeEventListener('click', cancelHandler);
    };
    
    const cancelHandler = () => {
        modal.classList.add('hidden');
        modalConfirm.removeEventListener('click', confirmHandler);
        modalCancel.removeEventListener('click', cancelHandler);
    };
    
    modalConfirm.addEventListener('click', confirmHandler);
    modalCancel.addEventListener('click', cancelHandler);
    
    // 点击背景关闭
    modal.addEventListener('click', function(e) {
        if (e.target === modal) {
            cancelHandler();
        }
    });
}

// 绑定状态切换事件
function bindStatusToggleEvents() {
    $('.status-toggle-btn').off('click').on('click', function(e) {
        e.preventDefault();
        e.stopPropagation();
        
        const $btn = $(this);
        const taskId = $btn.data('task-id');
        const currentStatus = $btn.data('current-status');
        
        // 禁用按钮并添加loading效果
        $btn.prop('disabled', true);
        
        // 切换状态
        toggleTaskStatus(taskId, currentStatus).finally(() => {
            $btn.prop('disabled', false);
        });
    });
}

// 切换任务状态
async function toggleTaskStatus(taskId, currentStatus) {
    const newStatus = currentStatus === 'active' ? 'inactive' : 'active';
    
    try {
        await Utils.api.put(`/api/tasks/${taskId}`, { status: newStatus });
        Utils.showToast(`任务已${newStatus === 'active' ? '激活' : '停用'}`, 'success');
        loadTasks();
    } catch (error) {
        console.error('Failed to toggle task status:', error);
        Utils.showToast('切换任务状态失败: ' + error.message, 'error');
    }
}
