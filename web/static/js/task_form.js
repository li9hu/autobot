// 任务表单页面 JavaScript

let isEditMode = false;
let taskId = null;
let editor = null;

// 页面加载完成后初始化
$(document).ready(function() {
    initializeTaskForm();
    bindEvents();
    initializeTimeExclusion(); // 确保时间排除功能在主初始化时就准备好
});

// 初始化任务表单
function initializeTaskForm() {
    // 检查是否为编辑模式
    const pathParts = window.location.pathname.split('/');
    if (pathParts.includes('edit')) {
        isEditMode = true;
        taskId = pathParts[pathParts.indexOf('tasks') + 1];
    }
    
    console.log('Task form initialized', { isEditMode, taskId });
    
    // 设置代码编辑器
    setupCodeEditor();
    
    // 如果没有任务数据，使用默认模板
    if (!window.taskData && !isEditMode) {
        setDefaultTemplate();
    }
}

// 绑定事件
function bindEvents() {
    // 表单提交
    $('#taskForm').on('submit', handleFormSubmit);
    
    // 验证脚本按钮
    $('#validateBtn').on('click', validateScript);
    
    // 插入模板按钮
    $('#insertTemplateBtn').on('click', insertTemplate);
    
    // 重置按钮
    $('#resetBtn').on('click', resetForm);
    
    // 实时验证
    $('#taskName').on('blur', validateTaskName);
    $('#cronExpr').on('blur', validateCronExpression);
}

// 设置代码编辑器
function setupCodeEditor() {
    const editorElement = document.getElementById('scriptEditor');
    const textarea = document.getElementById('taskScript');
    
    // 使用 CodeMirror
    editor = CodeMirror(editorElement, {
        value: textarea.value || '',
        mode: 'python',
        theme: 'github',
        lineNumbers: true,
        lineWrapping: true,
        indentUnit: 4,
        tabSize: 4,
        indentWithTabs: false,
        autoCloseBrackets: true,
        matchBrackets: true,
        viewportMargin: Infinity
    });
    
    // 同步到 textarea
    editor.on('change', function() {
        textarea.value = editor.getValue();
    });
    
    // 设置默认高度
    editor.setSize(null, '300px');
}

// 设置默认模板
function setDefaultTemplate() {
    const template = `def main():
    """
    任务的主要执行函数
    在这里编写您的代码逻辑
    """
    print("Hello, AutoBot!")
    
    # 在这里添加您的代码
    # 例如：
    # import requests
    # response = requests.get('https://api.example.com/data')
    # print(response.json())`;
    
    if (editor) {
        editor.setValue(template);
    }
}

// 插入模板
function insertTemplate() {
    const templates = {
        'basic': `def main():
    """基础模板"""
    print("Hello, AutoBot!"),
    print({"bark_key":"value"})`,
        
        'web_request': `import requests

def main():
    """Web请求模板"""
    try:
        response = requests.get('https://api.example.com/data')
        response.raise_for_status()
        data = response.json()
        print(f"获取数据成功: {data}")
    except requests.RequestException as e:
        print(f"请求失败: {e}")`,
    };
    
    // 显示模板选择
    showTemplateModal(templates);
}

// 显示模板选择模态框
function showTemplateModal(templates) {
    const modalHtml = `
        <div id="templateModal" class="fixed inset-0 bg-black bg-opacity-50 z-50 flex items-center justify-center p-4">
            <div class="bg-white rounded-xl shadow-xl max-w-2xl w-full max-h-[80vh] overflow-hidden">
                <div class="p-6 border-b border-slate-200">
                    <h3 class="text-lg font-semibold text-slate-900">选择代码模板</h3>
                </div>
                <div class="p-6 overflow-y-auto">
                    <div class="grid grid-cols-1 gap-3">
                        ${Object.entries(templates).map(([key, code]) => `
                            <button type="button" 
                                    onclick="selectTemplate('${key}')" 
                                    class="text-left p-4 border border-slate-200 rounded-lg hover:border-blue-300 hover:bg-blue-50 transition-colors">
                                <h4 class="font-medium text-slate-900 mb-2">${getTemplateName(key)}</h4>
                                <pre class="text-xs text-slate-600 bg-slate-50 p-2 rounded overflow-x-auto">${code.substring(0, 100)}...</pre>
                            </button>
                        `).join('')}
                    </div>
                </div>
                <div class="p-6 border-t border-slate-200 flex justify-end">
                    <button onclick="closeTemplateModal()" 
                            class="px-4 py-2 border border-slate-300 rounded-lg text-slate-700 hover:bg-slate-50 transition-colors">
                        取消
                    </button>
                </div>
            </div>
        </div>
    `;
    
    $('body').append(modalHtml);
    
    // 存储模板数据
    window.currentTemplates = templates;
}

// 选择模板
function selectTemplate(key) {
    const templates = window.currentTemplates;
    if (templates && templates[key] && editor) {
        editor.setValue(templates[key]);
    }
    closeTemplateModal();
}

// 关闭模板模态框
function closeTemplateModal() {
    $('#templateModal').remove();
    delete window.currentTemplates;
}

// 获取模板名称
function getTemplateName(key) {
    const names = {
        'basic': '基础模板',
        'web_request': 'Web请求模板',
    };
    return names[key] || key;
}

// 验证脚本
async function validateScript() {
    const script = editor ? editor.getValue() : $('#taskScript').val();
    
    if (!script.trim()) {
        showValidationResult(false, '脚本内容不能为空');
        return;
    }
    
    // 显示验证中状态
    const btn = $('#validateBtn');
    const originalHtml = btn.html();
    btn.html('<i data-lucide="loader-2" class="w-4 h-4 animate-spin"></i> 验证中...');
    btn.prop('disabled', true);
    
    try {
        const response = await Utils.api.post('/api/validate-script', { script });
        
        if (response.valid) {
            showValidationResult(true, '脚本语法正确');
        } else {
            showValidationResult(false, response.error || '脚本语法错误');
        }
    } catch (error) {
        showValidationResult(false, '验证失败: ' + error.message);
    } finally {
        btn.html(originalHtml);
        btn.prop('disabled', false);
        lucide.createIcons();
    }
}

// 显示验证结果
function showValidationResult(isValid, message) {
    const resultHtml = `
        <div class="flex items-center gap-2 p-3 rounded-lg mt-3 ${isValid ? 'bg-green-50 text-green-800' : 'bg-red-50 text-red-800'}">
            <i data-lucide="${isValid ? 'check-circle' : 'x-circle'}" class="w-4 h-4"></i>
            <span class="text-sm">${message}</span>
        </div>
    `;
    
    // 移除之前的结果
    $('.validation-result').remove();
    
    // 添加新结果
    $(resultHtml).addClass('validation-result').insertAfter('#scriptEditor');
    lucide.createIcons();
    
    // 3秒后自动移除
    setTimeout(() => {
        $('.validation-result').fadeOut(300, function() {
            $(this).remove();
        });
    }, 3000);
}

// 表单提交处理
async function handleFormSubmit(e) {
    e.preventDefault();
    
    if (!validateForm()) {
        return;
    }
    
    const formData = getFormData();
    
    // 显示加载状态
    showLoadingOverlay(true);
    
    try {
        let response;
        if (isEditMode) {
            response = await Utils.api.put(`/api/tasks/${taskId}`, formData);
        } else {
            response = await Utils.api.post('/api/tasks', formData);
        }
        
        Utils.showToast(
            isEditMode ? '任务更新成功' : '任务创建成功', 
            'success'
        );
        
        // 跳转到任务列表
        setTimeout(() => {
            window.location.href = '/';
        }, 1000);
        
    } catch (error) {
        console.error('Failed to save task:', error);
        Utils.showToast(
            (isEditMode ? '更新' : '创建') + '任务失败: ' + error.message, 
            'error'
        );
    } finally {
        showLoadingOverlay(false);
    }
}

// 获取表单数据
function getFormData() {
    const data = {
        name: $('#taskName').val().trim(),
        description: $('#taskDescription').val().trim(),
        script: editor ? editor.getValue() : $('#taskScript').val(),
        cron_expr: $('#cronExpr').val().trim()
    };
    
    // 添加时间排除配置
    const timeExclusionConfig = getTimeExclusionConfig();
    data.time_exclusion_config = JSON.stringify(timeExclusionConfig);
    
    return data;
}

// 验证表单
function validateForm() {
    let isValid = true;
    
    // 清除之前的错误
    $('.invalid-feedback').addClass('hidden');
    $('input, textarea, select').removeClass('border-red-300');
    
    // 验证任务名称
    if (!validateTaskName()) {
        isValid = false;
    }
    
    // 验证Cron表达式
    if (!validateCronExpression()) {
        isValid = false;
    }
    
    // 验证脚本
    if (!validateTaskScript()) {
        isValid = false;
    }
    
    return isValid;
}

// 验证任务名称
function validateTaskName() {
    const input = $('#taskName');
    const value = input.val().trim();
    const feedback = input.siblings('.invalid-feedback');
    
    if (!value) {
        showFieldError(input, feedback, '任务名称不能为空');
        return false;
    }
    
    if (value.length > 100) {
        showFieldError(input, feedback, '任务名称长度不能超过100个字符');
        return false;
    }
    
    hideFieldError(input, feedback);
    return true;
}

// 验证Cron表达式
function validateCronExpression() {
    const input = $('#cronExpr');
    const value = input.val().trim();
    const feedback = input.siblings('.invalid-feedback');
    
    if (!value) {
        showFieldError(input, feedback, 'Cron表达式不能为空');
        return false;
    }
    
    // 验证Cron表达式格式（6位：秒 分 时 日 月 周）
    const parts = value.split(/\s+/);
    if (parts.length !== 6) {
        showFieldError(input, feedback, 'Cron表达式格式错误，应包含6个部分（秒 分 时 日 月 周）');
        return false;
    }
    
    // 简单的字符验证
    const validChars = /^[0-9\*\-\,\/\?LW#]+$/;
    for (let part of parts) {
        if (!validChars.test(part)) {
            showFieldError(input, feedback, 'Cron表达式包含无效字符');
            return false;
        }
    }
    
    hideFieldError(input, feedback);
    return true;
}

// 验证任务脚本
function validateTaskScript() {
    const value = editor ? editor.getValue() : $('#taskScript').val();
    const feedback = $('#taskScript').siblings('.invalid-feedback');
    
    if (!value.trim()) {
        showFieldError($('#scriptEditor'), feedback, '脚本内容不能为空');
        return false;
    }
    
    if (!value.includes('def main():')) {
        showFieldError($('#scriptEditor'), feedback, '脚本必须包含 main() 函数定义');
        return false;
    }
    
    hideFieldError($('#scriptEditor'), feedback);
    return true;
}

// 显示字段错误
function showFieldError(field, feedback, message) {
    field.addClass('border-red-300');
    feedback.text(message).removeClass('hidden');
}

// 隐藏字段错误
function hideFieldError(field, feedback) {
    field.removeClass('border-red-300');
    feedback.addClass('hidden');
}

// 重置表单
function resetForm() {
    Utils.showConfirm(
        '重置表单',
        '确定要重置表单吗？这将清除所有已填写的内容。',
        function() {
            document.getElementById('taskForm').reset();
            if (editor) {
                setDefaultTemplate();
            }
            // 清除错误状态
            $('.invalid-feedback').addClass('hidden');
            $('input, textarea, select').removeClass('border-red-300');
        }
    );
}

// 显示/隐藏加载覆盖层
function showLoadingOverlay(show) {
    const overlay = $('#loadingOverlay');
    if (show) {
        overlay.removeClass('hidden');
    } else {
        overlay.addClass('hidden');
    }
}

// 点击模态框背景关闭
$(document).on('click', '#templateModal', function(e) {
    if (e.target === this) {
        closeTemplateModal();
    }
});

// ================== Cron 表达式相关功能 ==================

// 设置Cron表达式（快捷按钮）
function setCronExpr(expression) {
    const input = $('#cronExpr');
    input.val(expression);
    
    // 触发验证
    validateCronExpression();
    
    // 显示设置成功的提示
    showCronExpressionPreview(expression);
}

// 显示Cron表达式预览
function showCronExpressionPreview(expression) {
    // 移除之前的预览
    $('.cron-preview').remove();
    
    // 计算下次执行时间（简化版）
    const description = getCronDescription(expression);
    
    const previewHtml = `
        <div class="cron-preview bg-blue-50 border border-blue-200 rounded-lg p-3 mt-2">
            <div class="flex items-center gap-2 text-blue-800">
                <i data-lucide="clock" class="w-4 h-4"></i>
                <span class="text-sm font-medium">${description}</span>
            </div>
        </div>
    `;
    
    $(previewHtml).insertAfter('#cronExpr');
    lucide.createIcons();
    
    // 3秒后自动移除
    setTimeout(() => {
        $('.cron-preview').fadeOut(300, function() {
            $(this).remove();
        });
    }, 3000);
}

// 获取Cron表达式的描述
function getCronDescription(expression) {
    const descriptions = {
        '0 * * * * *': '每分钟执行一次',
        '0 */5 * * * *': '每5分钟执行一次',
        '0 */10 * * * *': '每10分钟执行一次',
        '0 */15 * * * *': '每15分钟执行一次',
        '0 */30 * * * *': '每30分钟执行一次',
        '0 0 * * * *': '每小时执行一次',
        '0 0 */6 * * *': '每6小时执行一次',
        '0 0 */12 * * *': '每12小时执行一次',
        '0 0 0 * * *': '每天执行一次',
        '0 0 2 * * *': '每天凌晨2点执行',
        '0 0 9 * * 1-5': '工作日上午9点执行',
        '0 0 0 * * 0': '每周日执行',
        '0 0 0 1 * *': '每月1号执行'
    };
    
    return descriptions[expression] || '自定义执行周期';
}

// 显示Cron帮助器
function showCronHelper() {
    const modal = document.getElementById('cronHelperModal');
    modal.classList.remove('hidden');
    
    // 初始化构建器
    initializeCronBuilder();
    
    // 重新初始化图标
    lucide.createIcons();
}

// 关闭Cron帮助器
function closeCronHelper() {
    const modal = document.getElementById('cronHelperModal');
    modal.classList.add('hidden');
}

// 初始化Cron构建器
function initializeCronBuilder() {
    // 绑定更改事件
    $('#cronSeconds, #cronMinutes, #cronHours, #cronDays, #cronMonths, #cronWeekdays').on('change', updateCronBuilder);
    
    // 初始更新
    updateCronBuilder();
}

// 更新Cron构建器
function updateCronBuilder() {
    const seconds = $('#cronSeconds').val() || '0';
    const minutes = $('#cronMinutes').val() || '*';
    const hours = $('#cronHours').val() || '*';
    const days = $('#cronDays').val() || '*';
    const months = $('#cronMonths').val() || '*';
    const weekdays = $('#cronWeekdays').val() || '*';
    
    const cronExpression = `${seconds} ${minutes} ${hours} ${days} ${months} ${weekdays}`;
    $('#generatedCron').text(cronExpression);
}

// 应用构建器生成的Cron表达式
function applyCronFromBuilder() {
    const expression = $('#generatedCron').text();
    $('#cronExpr').val(expression);
    
    // 关闭模态框
    closeCronHelper();
    
    // 验证表达式
    validateCronExpression();
    
    // 显示成功提示
    // Utils.showToast('Cron表达式已应用', 'success', 2000);
}

// 点击模态框背景关闭Cron帮助器
$(document).on('click', '#cronHelperModal', function(e) {
    if (e.target === this) {
        closeCronHelper();
    }
});

// ESC键关闭模态框
$(document).on('keydown', function(e) {
    if (e.key === 'Escape') {
        if (!$('#cronHelperModal').hasClass('hidden')) {
            closeCronHelper();
        }
        if (!$('#templateModal').hasClass('hidden')) {
            closeTemplateModal();
        }
    }
});

// =============================================================================
// 时间排除功能
// =============================================================================

let timeExclusionRules = [];

// 初始化时间排除功能
function initializeTimeExclusion() {
    // 绑定启用复选框事件
    $('#timeExclusionEnabled').on('change', function() {
        const isEnabled = $(this).is(':checked');
        const rulesContainer = $('#timeExclusionRules');
        
        if (isEnabled) {
            // 启用时显示规则区域
            rulesContainer.removeClass('hidden');
            rulesContainer.removeClass('opacity-50');
        } else {
            // 禁用时：如果有规则就显示但半透明，没有规则就隐藏
            if (timeExclusionRules.length > 0) {
                rulesContainer.removeClass('hidden');
                rulesContainer.addClass('opacity-50');
            } else {
                rulesContainer.addClass('hidden');
                rulesContainer.removeClass('opacity-50');
            }
        }
    });

    // 绑定添加规则按钮
    $('#addRuleBtn').on('click', addTimeExclusionRule);

    // 如果是编辑模式，延迟加载时间排除配置，确保taskData已经设置
    if (isEditMode) {
        // 延迟执行，确保DOM和数据都准备好
        setTimeout(() => {
            if (window.taskData && window.taskData.time_exclusion_config) {
                loadTimeExclusionConfig(window.taskData.time_exclusion_config);
            }
        }, 100);
    }
}

// 加载时间排除配置
function loadTimeExclusionConfig(configJson) {
    try {
        const config = JSON.parse(configJson);
        
        // 设置启用状态
        $('#timeExclusionEnabled').prop('checked', config.enabled);
        
        // 如果启用或者有已保存的规则，都显示规则区域
        if (config.enabled || (config.exclusion_rules && config.exclusion_rules.length > 0)) {
            $('#timeExclusionRules').removeClass('hidden');
        }

        // 加载规则（不论启用状态如何都加载）
        timeExclusionRules = config.exclusion_rules || [];
        renderTimeExclusionRules();
        
    } catch (error) {
        console.error('Failed to parse time exclusion config:', error);
    }
}

// 添加时间排除规则
function addTimeExclusionRule() {
    const rule = {
        type: 'daily',
        start_time: '22:00',
        end_time: '06:00',
        weekdays: [],
        start_date: '',
        end_date: '',
        _isEditing: true,  // 新规则默认进入编辑模式
        _isNew: true       // 标记为新规则
    };
    
    timeExclusionRules.push(rule);
    
    // 添加规则时自动启用时间排除并显示规则区域
    const enabledCheckbox = $('#timeExclusionEnabled');
    const rulesContainer = $('#timeExclusionRules');
    
    if (!enabledCheckbox.is(':checked')) {
        enabledCheckbox.prop('checked', true);
    }
    rulesContainer.removeClass('hidden opacity-50');
    
    renderTimeExclusionRules();
}

// 渲染时间排除规则列表
function renderTimeExclusionRules() {
    const rulesList = $('#rulesList');
    rulesList.empty();

    if (timeExclusionRules.length === 0) {
        rulesList.html('<p class="text-sm text-slate-500 text-center py-8">暂无排除规则，点击"添加规则"开始配置</p>');
        return;
    }

    timeExclusionRules.forEach((rule, index) => {
        // 检查规则是否处于编辑状态
        const isEditing = rule._isEditing === true;
        const ruleHtml = createRuleElement(rule, index, isEditing);
        rulesList.append(ruleHtml);
    });

    // 重新绑定事件
    bindRuleEvents();
    
    // 重新渲染图标
    lucide.createIcons();
}

// 创建规则元素
function createRuleElement(rule, index, isEditing = false) {
    const weekdays = ['周日', '周一', '周二', '周三', '周四', '周五', '周六'];
    
    // 根据规则类型选择图标
    const getTypeIcon = (type) => {
        switch(type) {
            case 'daily': return 'clock';
            case 'weekly': return 'calendar-days';
            case 'date_range': return 'calendar-range';
            default: return 'clock';
        }
    };

    // 获取规则类型的显示名称
    const getTypeName = (type) => {
        switch(type) {
            case 'daily': return 'daily';
            case 'weekly': return 'weekly';
            case 'date_range': return 'date';
            default: return 'daily';
        }
    };

    // 格式化显示文本
    const formatTimeRange = (startTime, endTime) => {
        return `${startTime} - ${endTime}`;
    };

    const formatWeekdays = (weekdaysList) => {
        if (!weekdaysList || weekdaysList.length === 0) return '';
        return weekdaysList.map(day => weekdays[day]).join('、');
    };

    const formatDateRange = (startDate, endDate) => {
        if (!startDate || !endDate) return '';
        return `${startDate} 至 ${endDate}`;
    };

    if (isEditing) {
        // 编辑模式
        return `
            <div class="bg-white border-2 border-blue-200 rounded-lg p-3 rule-item shadow-md" data-index="${index}" data-editing="true">
                <!-- 编辑模式头部 -->
                <div class="flex items-center justify-between mb-3">
                    <div class="flex items-center space-x-1">
                        <i data-lucide="edit-3" class="w-3.5 h-3.5 text-blue-600"></i>
                        <span class="text-xs font-medium text-slate-700">编辑</span>
                    </div>
                    <div class="flex items-center space-x-1">
                        <button type="button" class="confirm-rule inline-flex items-center justify-center p-1 bg-green-50 text-green-700 rounded hover:bg-green-100 transition-colors" title="确认">
                            <i data-lucide="check" class="w-3 h-3"></i>
                        </button>
                        <button type="button" class="cancel-rule inline-flex items-center justify-center p-1 bg-gray-50 text-gray-700 rounded hover:bg-gray-100 transition-colors" title="取消">
                            <i data-lucide="x" class="w-3 h-3"></i>
                        </button>
                    </div>
                </div>

                <!-- 规则配置 -->
                <div class="space-y-2">
                    <!-- 规则类型 -->
                    <div>
                        <label class="block text-xs font-medium text-slate-700 mb-1">类型</label>
                        <select class="rule-type w-full px-2 py-1.5 border border-slate-300 rounded text-xs focus:ring-1 focus:ring-blue-500 focus:border-blue-500 outline-none transition-colors bg-white">
                            <option value="daily" ${rule.type === 'daily' ? 'selected' : ''}>每日</option>
                            <option value="weekly" ${rule.type === 'weekly' ? 'selected' : ''}>每周</option>
                            <option value="date_range" ${rule.type === 'date_range' ? 'selected' : ''}>日期范围</option>
                        </select>
                    </div>

                    <!-- 时间设置 -->
                    <div class="grid grid-cols-2 gap-2">
                        <div>
                            <label class="block text-xs font-medium text-slate-700 mb-1">开始</label>
                            <input type="time" 
                                   class="rule-start-time w-full px-2 py-1.5 border border-slate-300 rounded text-xs focus:ring-1 focus:ring-blue-500 focus:border-blue-500 outline-none transition-colors"
                                   value="${rule.start_time}">
                        </div>
                        <div>
                            <label class="block text-xs font-medium text-slate-700 mb-1">结束</label>
                            <input type="time" 
                                   class="rule-end-time w-full px-2 py-1.5 border border-slate-300 rounded text-xs focus:ring-1 focus:ring-blue-500 focus:border-blue-500 outline-none transition-colors"
                                   value="${rule.end_time}">
                        </div>
                    </div>

                    <!-- 周几设置（仅weekly类型显示） -->
                    <div class="weekdays-settings ${rule.type !== 'weekly' ? 'hidden' : ''}">
                        <label class="block text-xs font-medium text-slate-700 mb-1">星期</label>
                        <div class="grid grid-cols-7 gap-1">
                            ${weekdays.map((day, dayIndex) => `
                                <label class="flex flex-col items-center p-1 border border-slate-200 rounded cursor-pointer hover:bg-slate-50 transition-colors text-xs ${rule.weekdays && rule.weekdays.includes(dayIndex) ? 'bg-blue-50 border-blue-300' : ''}">
                                    <input type="checkbox" 
                                           class="rule-weekday w-2.5 h-2.5 text-blue-600 bg-slate-100 border-slate-300 rounded focus:ring-blue-500 focus:ring-1 mb-0.5" 
                                           value="${dayIndex}"
                                           ${rule.weekdays && rule.weekdays.includes(dayIndex) ? 'checked' : ''}>
                                    <span class="text-xs font-medium text-slate-700">${day}</span>
                                </label>
                            `).join('')}
                        </div>
                    </div>

                    <!-- 日期范围设置（仅date_range类型显示） -->
                    <div class="date-range-settings ${rule.type !== 'date_range' ? 'hidden' : ''} grid grid-cols-2 gap-2">
                        <div>
                            <label class="block text-xs font-medium text-slate-700 mb-1">开始日期</label>
                            <input type="date" 
                                   class="rule-start-date w-full px-2 py-1.5 border border-slate-300 rounded text-xs focus:ring-1 focus:ring-blue-500 focus:border-blue-500 outline-none transition-colors"
                                   value="${rule.start_date || ''}">
                        </div>
                        <div>
                            <label class="block text-xs font-medium text-slate-700 mb-1">结束日期</label>
                            <input type="date" 
                                   class="rule-end-date w-full px-2 py-1.5 border border-slate-300 rounded text-xs focus:ring-1 focus:ring-blue-500 focus:border-blue-500 outline-none transition-colors"
                                   value="${rule.end_date || ''}">
                        </div>
                    </div>
                </div>
            </div>
        `;
    } else {
        // 展示模式
        return `
            <div class="bg-white border border-slate-200 rounded-lg p-3 rule-item shadow-sm hover:shadow-md transition-shadow" data-index="${index}" data-editing="false">
                <!-- 展示模式头部 -->
                <div class="flex items-center justify-between mb-2">
                    <div class="flex items-center space-x-2">
                        <i data-lucide="${getTypeIcon(rule.type)}" class="w-3.5 h-3.5 text-blue-600"></i>
                        <span class="text-xs font-medium text-slate-700">${getTypeName(rule.type)}</span>
                    </div>
                    <div class="flex items-center space-x-1">
                        <button type="button" class="edit-rule inline-flex items-center justify-center p-1 bg-blue-50 text-blue-700 rounded hover:bg-blue-100 transition-colors" title="编辑">
                            <i data-lucide="edit-2" class="w-3 h-3"></i>
                        </button>
                        <button type="button" class="remove-rule inline-flex items-center justify-center p-1 bg-red-50 text-red-700 rounded hover:bg-red-100 transition-colors" title="删除">
                            <i data-lucide="trash-2" class="w-3 h-3"></i>
                        </button>
                    </div>
                </div>

                <!-- 规则详情展示 -->
                <div class="space-y-1">
                    <!-- 时间信息 -->
                    <div class="flex items-center space-x-1 text-xs text-slate-600">
                        <i data-lucide="clock" class="w-3 h-3 text-slate-400"></i>
                        <span>${formatTimeRange(rule.start_time, rule.end_time)}</span>
                    </div>

                    <!-- 周几信息（仅weekly类型显示） -->
                    ${rule.type === 'weekly' && rule.weekdays && rule.weekdays.length > 0 ? `
                        <div class="flex items-center space-x-1 text-xs text-slate-600">
                            <i data-lucide="calendar-days" class="w-3 h-3 text-slate-400"></i>
                            <span>${formatWeekdays(rule.weekdays)}</span>
                        </div>
                    ` : ''}

                    <!-- 日期范围信息（仅date_range类型显示） -->
                    ${rule.type === 'date_range' && rule.start_date && rule.end_date ? `
                        <div class="flex items-center space-x-1 text-xs text-slate-600">
                            <i data-lucide="calendar-range" class="w-3 h-3 text-slate-400"></i>
                            <span>${formatDateRange(rule.start_date, rule.end_date)}</span>
                        </div>
                    ` : ''}
                </div>
            </div>
        `;
    }
}

// 绑定规则事件
function bindRuleEvents() {
    // 编辑规则按钮
    $('.edit-rule').on('click', function() {
        const index = parseInt($(this).closest('.rule-item').data('index'));
        timeExclusionRules[index]._isEditing = true;
        renderTimeExclusionRules();
    });

    // 确认规则按钮
    $('.confirm-rule').on('click', function() {
        const index = parseInt($(this).closest('.rule-item').data('index'));
        const ruleItem = $(this).closest('.rule-item');
        
        // 验证必填字段
        const startTime = ruleItem.find('.rule-start-time').val();
        const endTime = ruleItem.find('.rule-end-time').val();
        if (!startTime || !endTime) {
            alert('请设置完整的时间段');
            return;
        }
        
        // 确保规则对象存在
        if (!timeExclusionRules[index]) {
            timeExclusionRules[index] = {};
        }
        
        // 保存表单中的所有值到规则对象
        timeExclusionRules[index].type = ruleItem.find('.rule-type').val();
        timeExclusionRules[index].start_time = startTime;
        timeExclusionRules[index].end_time = endTime;
        
        // 获取星期选择（仅weekly类型）
        if (timeExclusionRules[index].type === 'weekly') {
            const weekdays = [];
            ruleItem.find('.rule-weekday:checked').each(function() {
                weekdays.push(parseInt($(this).val()));
            });
            timeExclusionRules[index].weekdays = weekdays;
        }
        
        // 获取日期范围（仅date_range类型）
        if (timeExclusionRules[index].type === 'date_range') {
            const startDate = ruleItem.find('.rule-start-date').val();
            const endDate = ruleItem.find('.rule-end-date').val();
            if (!startDate || !endDate) {
                alert('请设置完整的日期范围');
                return;
            }
            timeExclusionRules[index].start_date = startDate;
            timeExclusionRules[index].end_date = endDate;
        }
        
        // 保存并退出编辑模式
        delete timeExclusionRules[index]._isEditing;
        delete timeExclusionRules[index]._isNew;  // 移除新规则标记
        renderTimeExclusionRules();
    });

    // 取消编辑按钮
    $('.cancel-rule').on('click', function() {
        const index = parseInt($(this).closest('.rule-item').data('index'));
        
        // 如果是新规则，则删除
        if (timeExclusionRules[index]._isEditing && timeExclusionRules[index]._isNew) {
            timeExclusionRules.splice(index, 1);
        } else {
            // 恢复编辑状态
            delete timeExclusionRules[index]._isEditing;
        }
        
        renderTimeExclusionRules();
    });

    // 删除规则按钮
    $('.remove-rule').on('click', function() {
        const index = parseInt($(this).closest('.rule-item').data('index'));
        const rule = timeExclusionRules[index];
        const ruleTypeNames = {
            'daily': '每日',
            'weekly': '每周',
            'date_range': '日期范围'
        };
        const ruleDescription = `${ruleTypeNames[rule.type] || rule.type} ${rule.start_time}-${rule.end_time}`;
        
        Utils.showConfirm(
            '删除时间排除规则',
            `确定要删除规则 "<strong>${ruleDescription}</strong>" 吗？<br><br>此操作不可撤销。`,
            function() {
                timeExclusionRules.splice(index, 1);
                renderTimeExclusionRules();
                Utils.showToast('时间排除规则已删除', 'success');
            }
        );
    });

    // 规则类型改变
    $('.rule-type').on('change', function() {
        const ruleItem = $(this).closest('.rule-item');
        const index = parseInt(ruleItem.data('index'));
        const type = $(this).val();
        
        timeExclusionRules[index].type = type;
        
        // 显示/隐藏相应的设置
        const weekdaysSettings = ruleItem.find('.weekdays-settings');
        const dateRangeSettings = ruleItem.find('.date-range-settings');
        
        if (type === 'weekly') {
            weekdaysSettings.removeClass('hidden');
            dateRangeSettings.addClass('hidden');
        } else if (type === 'date_range') {
            weekdaysSettings.addClass('hidden');
            dateRangeSettings.removeClass('hidden');
        } else {
            weekdaysSettings.addClass('hidden');
            dateRangeSettings.addClass('hidden');
        }
    });

    // 规则字段改变事件（仅在编辑模式下）
    $('.rule-name').on('input', function() {
        const index = parseInt($(this).closest('.rule-item').data('index'));
        if (timeExclusionRules[index]) {
            timeExclusionRules[index].name = $(this).val();
        }
    });

    $('.rule-start-time').on('change', function() {
        const index = parseInt($(this).closest('.rule-item').data('index'));
        if (timeExclusionRules[index]) {
            timeExclusionRules[index].start_time = $(this).val();
        }
    });

    $('.rule-end-time').on('change', function() {
        const index = parseInt($(this).closest('.rule-item').data('index'));
        if (timeExclusionRules[index]) {
            timeExclusionRules[index].end_time = $(this).val();
        }
    });

    $('.rule-start-date').on('change', function() {
        const index = parseInt($(this).closest('.rule-item').data('index'));
        if (timeExclusionRules[index]) {
            timeExclusionRules[index].start_date = $(this).val();
        }
    });

    $('.rule-end-date').on('change', function() {
        const index = parseInt($(this).closest('.rule-item').data('index'));
        if (timeExclusionRules[index]) {
            timeExclusionRules[index].end_date = $(this).val();
        }
    });

    $('.rule-weekday').on('change', function() {
        const index = parseInt($(this).closest('.rule-item').data('index'));
        if (timeExclusionRules[index]) {
            const weekdays = [];
            $(this).closest('.rule-item').find('.rule-weekday:checked').each(function() {
                weekdays.push(parseInt($(this).val()));
            });
            timeExclusionRules[index].weekdays = weekdays;
        }
    });
}

// 获取时间排除配置
function getTimeExclusionConfig() {
    const enabled = $('#timeExclusionEnabled').is(':checked');
    
    // 清理规则数据，移除内部状态标志
    const cleanRules = timeExclusionRules.map(rule => {
        const cleanRule = { ...rule };
        delete cleanRule._isEditing;  // 移除编辑状态标志
        return cleanRule;
    });
    
    const config = {
        enabled: enabled,
        exclusion_rules: cleanRules  // 始终保存规则，不管是否启用
    };
    
    // 添加调试日志
    console.log('获取时间排除配置:', config);
    
    return config;
}

// 时间排除功能已在主初始化流程中调用