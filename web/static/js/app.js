// 全局应用 JavaScript

// 应用配置
const APP_CONFIG = {
    PAGE_SIZE: 12,
    API_BASE_URL: '',
    TOAST_DURATION: 3000
};

// 工具类
const Utils = {
    // API 请求工具
    api: {
        async request(url, options = {}) {
            const config = {
                headers: {
                    'Content-Type': 'application/json',
                    ...options.headers
                },
                ...options
            };

            if (config.body && typeof config.body === 'object') {
                config.body = JSON.stringify(config.body);
            }

            try {
                const response = await fetch(APP_CONFIG.API_BASE_URL + url, config);
                
                // 检查是否为401未授权错误
                if (response.status === 401) {
                    // 如果是API请求返回401，跳转到登录页面
                    if (url.startsWith('/api/')) {
                        window.location.href = '/login';
                        return;
                    }
                }
                
                if (!response.ok) {
                    let errorMessage = `HTTP ${response.status}`;
                    try {
                        // 克隆响应以避免body被重复读取
                        const errorResponse = response.clone();
                        const errorData = await errorResponse.json();
                        errorMessage = errorData.error || errorMessage;
                    } catch (e) {
                        try {
                            const errorText = await response.text();
                            errorMessage = errorText || errorMessage;
                        } catch (textError) {
                            // 如果都失败了，使用默认错误消息
                            errorMessage = `HTTP ${response.status}`;
                        }
                    }
                    throw new Error(errorMessage);
                }

                const contentType = response.headers.get('content-type');
                if (contentType && contentType.includes('application/json')) {
                    return await response.json();
                }
                
                return await response.text();
            } catch (error) {
                console.error('API request failed:', error);
                throw error;
            }
        },

        async get(url, params = {}) {
            const searchParams = new URLSearchParams(params).toString();
            const finalUrl = searchParams ? `${url}?${searchParams}` : url;
            return this.request(finalUrl, { method: 'GET' });
        },

        async post(url, data = {}) {
            return this.request(url, {
                method: 'POST',
                body: data
            });
        },

        async put(url, data = {}) {
            return this.request(url, {
                method: 'PUT',
                body: data
            });
        },

        async delete(url) {
            return this.request(url, { method: 'DELETE' });
        }
    },

    // 格式化日期时间
    formatDateTime(dateString) {
        if (!dateString) return '-';
        
        try {
            const date = new Date(dateString);
            return date.toLocaleString('zh-CN', {
                year: 'numeric',
                month: '2-digit',
                day: '2-digit',
                hour: '2-digit',
                minute: '2-digit',
                second: '2-digit'
            });
        } catch (error) {
            return dateString;
        }
    },

    // 格式化相对时间
    formatRelativeTime(dateString) {
        if (!dateString) return '-';
        
        try {
            const date = new Date(dateString);
            const now = new Date();
            const diff = now - date;
            
            const seconds = Math.floor(diff / 1000);
            const minutes = Math.floor(seconds / 60);
            const hours = Math.floor(minutes / 60);
            const days = Math.floor(hours / 24);
            
            if (days > 0) {
                return `${days}天前`;
            } else if (hours > 0) {
                return `${hours}小时前`;
            } else if (minutes > 0) {
                return `${minutes}分钟前`;
            } else {
                return '刚刚';
            }
        } catch (error) {
            return this.formatDateTime(dateString);
        }
    },

    // 格式化持续时间
    formatDuration(milliseconds) {
        if (!milliseconds || milliseconds < 0) return '-';
        
        const seconds = Math.floor(milliseconds / 1000);
        const minutes = Math.floor(seconds / 60);
        const hours = Math.floor(minutes / 60);
        
        if (hours > 0) {
            return `${hours}h ${minutes % 60}m ${seconds % 60}s`;
        } else if (minutes > 0) {
            return `${minutes}m ${seconds % 60}s`;
        } else {
            return `${seconds}s`;
        }
    },

    // 显示Toast消息
    showToast(message, type = 'info', duration = APP_CONFIG.TOAST_DURATION) {
        // 移除现有的toast
        const existingToast = document.getElementById('app-toast');
        if (existingToast) {
            existingToast.remove();
        }

        // 创建新的toast
        const toast = document.createElement('div');
        toast.id = 'app-toast';
        toast.className = `fixed top-4 right-4 z-50 max-w-sm w-full transform transition-all duration-300 ease-in-out`;
        
        const bgColor = {
            'success': 'bg-green-50 border-green-200 text-green-800',
            'error': 'bg-red-50 border-red-200 text-red-800',
            'warning': 'bg-amber-50 border-amber-200 text-amber-800',
            'info': 'bg-blue-50 border-blue-200 text-blue-800'
        }[type] || 'bg-blue-50 border-blue-200 text-blue-800';

        const iconName = {
            'success': 'check-circle',
            'error': 'x-circle',
            'warning': 'alert-triangle',
            'info': 'info'
        }[type] || 'info';

        toast.innerHTML = `
            <div class="border rounded-xl p-4 shadow-lg ${bgColor}">
                <div class="flex items-start">
                    <i data-lucide="${iconName}" class="w-5 h-5 mt-0.5 mr-3 flex-shrink-0"></i>
                    <div class="flex-1 min-w-0">
                        <p class="text-sm font-medium">${message}</p>
                    </div>
                    <button onclick="this.closest('#app-toast').remove()" class="ml-3 text-current opacity-70 hover:opacity-100">
                        <i data-lucide="x" class="w-4 h-4"></i>
                    </button>
                </div>
            </div>
        `;

        // 添加到页面
        document.body.appendChild(toast);
        
        // 初始化图标
        lucide.createIcons();

        // 显示动画
        setTimeout(() => {
            toast.style.transform = 'translateX(0)';
            toast.style.opacity = '1';
        }, 10);

        // 自动移除
        if (duration > 0) {
            setTimeout(() => {
                if (toast.parentNode) {
                    toast.style.transform = 'translateX(100%)';
                    toast.style.opacity = '0';
                    setTimeout(() => {
                        if (toast.parentNode) {
                            toast.remove();
                        }
                    }, 300);
                }
            }, duration);
        }
    },

    // HTML转义
    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    },

    // 截断文本
    truncateText(text, maxLength = 100) {
        if (!text || text.length <= maxLength) {
            return text;
        }
        return text.substring(0, maxLength) + '...';
    },

    // 复制到剪贴板
    async copyToClipboard(text) {
        try {
            await navigator.clipboard.writeText(text);
            this.showToast('已复制到剪贴板', 'success', 1500);
        } catch (error) {
            console.error('Failed to copy text:', error);
            this.showToast('复制失败', 'error');
        }
    },

    // 下载文件
    downloadFile(content, filename, mimeType = 'text/plain') {
        const blob = new Blob([content], { type: mimeType });
        const url = URL.createObjectURL(blob);
        const link = document.createElement('a');
        link.href = url;
        link.download = filename;
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
        URL.revokeObjectURL(url);
    },

    // 防抖函数
    debounce(func, wait) {
        let timeout;
        return function executedFunction(...args) {
            const later = () => {
                clearTimeout(timeout);
                func(...args);
            };
            clearTimeout(timeout);
            timeout = setTimeout(later, wait);
        };
    },

    // 节流函数
    throttle(func, limit) {
        let inThrottle;
        return function() {
            const args = arguments;
            const context = this;
            if (!inThrottle) {
                func.apply(context, args);
                inThrottle = true;
                setTimeout(() => inThrottle = false, limit);
            }
        };
    },

    // 获取状态徽章
    getStatusBadge(status) {
        if (status === 'active') {
            return `<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
                        <span class="w-1.5 h-1.5 bg-green-400 rounded-full mr-1.5"></span>
                        已激活
                    </span>`;
        } else if (status === 'success') {
            return `<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
                        <span class="w-1.5 h-1.5 bg-green-400 rounded-full mr-1.5"></span>
                        成功
                    </span>`;
        } else if (status === 'execution_failed') {
            return `<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-800">
                        <span class="w-1.5 h-1.5 bg-red-400 rounded-full mr-1.5"></span>
                        执行失败
                    </span>`;
        } else if (status === 'script_failed') {
            return `<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-orange-100 text-orange-800">
                        <span class="w-1.5 h-1.5 bg-orange-400 rounded-full mr-1.5"></span>
                        脚本错误
                    </span>`;
        } else if (status === 'failed') {
            return `<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-800">
                        <span class="w-1.5 h-1.5 bg-red-400 rounded-full mr-1.5"></span>
                        失败
                    </span>`;
        } else if (status === 'running') {
            return `<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800">
                        <span class="w-1.5 h-1.5 bg-blue-400 rounded-full mr-1.5"></span>
                        运行中
                    </span>`;
        } else {
            return `<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800">
                        <span class="w-1.5 h-1.5 bg-gray-400 rounded-full mr-1.5"></span>
                        ${status || '未知'}
                    </span>`;
        }
    },

    // 显示确认对话框
    showConfirm(title, content, onConfirm) {
        const modal = document.getElementById('confirmModal');
        const modalTitle = document.getElementById('modalTitle');
        const modalBody = document.getElementById('modalBody');
        const modalConfirm = document.getElementById('modalConfirm');
        const modalCancel = document.getElementById('modalCancel');
        
        // 检查必要的元素是否存在
        if (!modal || !modalTitle || !modalBody || !modalConfirm || !modalCancel) {
            console.error('确认模态框的必要元素未找到');
            return;
        }
        
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
};

// 全局错误处理
window.addEventListener('error', function(event) {
    console.error('Global error:', event.error);
    Utils.showToast('系统出现错误，请稍后重试', 'error');
});

window.addEventListener('unhandledrejection', function(event) {
    console.error('Unhandled promise rejection:', event.reason);
    Utils.showToast('网络请求失败，请检查网络连接', 'error');
});

// 页面加载完成后的通用初始化
document.addEventListener('DOMContentLoaded', function() {
    // 初始化工具提示
    initializeTooltips();
    
    // 初始化快捷键
    initializeKeyboardShortcuts();
    
    console.log('App initialized');
});

// 初始化工具提示
function initializeTooltips() {
    // 可以在这里添加工具提示的初始化代码
}

// 初始化快捷键
function initializeKeyboardShortcuts() {
    document.addEventListener('keydown', function(event) {
        // Ctrl/Cmd + R: 刷新页面数据
        if ((event.ctrlKey || event.metaKey) && event.key === 'r') {
            event.preventDefault();
            if (typeof loadTasks === 'function') {
                loadTasks();
            } else if (typeof loadLogs === 'function') {
                loadLogs();
            }
        }
        
        // ESC: 关闭模态框
        if (event.key === 'Escape') {
            const modals = document.querySelectorAll('[id$="Modal"]:not(.hidden)');
            modals.forEach(modal => {
                modal.classList.add('hidden');
            });
        }
    });
}

// 导出到全局作用域
window.Utils = Utils;
window.APP_CONFIG = APP_CONFIG;