// 身份认证相关的JavaScript功能

// 认证工具类
const Auth = {
    // 初始化用户菜单功能
    initUserMenu() {
        const userMenuButton = document.getElementById('user-menu-button');
        const userDropdown = document.getElementById('user-dropdown');
        const logoutBtn = document.getElementById('logout-btn');
        const usernameDisplay = document.getElementById('username-display');

        if (!userMenuButton || !userDropdown || !logoutBtn || !usernameDisplay) {
            console.warn('User menu elements not found');
            return;
        }

        // Toggle user dropdown
        userMenuButton.addEventListener('click', function(e) {
            e.stopPropagation();
            userDropdown.classList.toggle('hidden');
        });

        // Close dropdown when clicking outside
        document.addEventListener('click', function() {
            userDropdown.classList.add('hidden');
        });

        // Prevent dropdown from closing when clicking inside
        userDropdown.addEventListener('click', function(e) {
            e.stopPropagation();
        });

        // Logout functionality
        logoutBtn.addEventListener('click', async function() {
            try {
                const response = await Utils.api.request('/api/logout', {
                    method: 'POST'
                });
                
                if (response) {
                    Utils.showToast('已成功退出登录', 'success');
                    setTimeout(() => {
                        window.location.href = '/login';
                    }, 1000);
                }
            } catch (error) {
                const errorMessage = error && error.message ? error.message : '未知错误';
                Utils.showToast('退出登录失败: ' + errorMessage, 'error');
            }
        });

        // Load current user info
        this.loadCurrentUser();
    },

    // 加载当前用户信息
    async loadCurrentUser() {
        try {
            const response = await Utils.api.request('/api/me');
            if (response && response.user) {
                const usernameDisplay = document.getElementById('username-display');
                if (usernameDisplay) {
                    usernameDisplay.textContent = response.user.username;
                }
                return response.user;
            }
        } catch (error) {
            console.error('Failed to load user info:', error);
            // If failed to load user info, redirect to login
            window.location.href = '/login';
        }
        return null;
    },

    // 检查用户是否已登录
    async checkAuth() {
        try {
            const response = await Utils.api.request('/api/me');
            return response && response.user;
        } catch (error) {
            return false;
        }
    },

    // 登出
    async logout() {
        try {
            await Utils.api.request('/api/logout', {
                method: 'POST'
            });
            window.location.href = '/login';
        } catch (error) {
            console.error('Logout failed:', error);
            // Force redirect even if logout API fails
            window.location.href = '/login';
        }
    }
};

// 页面加载完成后自动初始化
document.addEventListener('DOMContentLoaded', function() {
    // 如果不是登录页面，则初始化用户菜单
    if (!window.location.pathname.includes('/login')) {
        Auth.initUserMenu();
    }
});
