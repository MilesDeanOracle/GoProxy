<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Copy, FilePlus2, Info, Network, Pencil, Plus, Route, Save, Trash2, Copy as CopyIcon } from 'lucide-vue-next'
import {
  NButton,
  NDrawer,
  NDrawerContent,
  NIcon,
  NInput,
  NInputNumber,
  NSelect,
  NSpace,
  NSpin,
  NSwitch,
  NTooltip,
  useDialog,
  useMessage
} from 'naive-ui'
import {
  createRouteFile,
  deleteRouteFile,
  getNetworkInterfaces,
  listRouteFiles,
  loadRouteFile,
  saveRouteFile,
  setActiveRouteFile
} from '../backend/api'
import { useConfigStore } from '../stores/config'
import { useServerStore } from '../stores/server'
import type { NetworkInterface, RouteFileInfo, RouteRule, RouteRuleSet } from '../types'
import { friendlyError } from '../utils/errors'

type RuleDraft = RouteRule & { targetText: string }

const config = useConfigStore()
const server = useServerStore()
const message = useMessage()
const dialog = useDialog()

const files = ref<RouteFileInfo[]>([])
const ruleSet = ref<RouteRuleSet | null>(null)
const interfaces = ref<NetworkInterface[]>([])
const loading = ref(false)
const saving = ref(false)
const error = ref('')
const modalVisible = ref(false)
const editingIndex = ref(-1)
const draft = ref<RuleDraft>(emptyRule())
const deleteTarget = ref('')

const activeFile = computed(() => config.draft?.route.activeFile ?? 'default.rule')
const fileOptions = computed(() => files.value.map((file) => ({ label: file.name, value: file.name })))
const interfaceOptions = computed(() =>
  interfaces.value.map((item) => ({
    label: `${item.name}${item.addresses.length > 0 ? ` / ${item.addresses.join(', ')}` : ''}`,
    value: item.name
  }))
)
const deletableFileOptions = computed(() =>
  files.value.filter((file) => file.name !== activeFile.value && file.name !== 'default.rule').map((file) => ({ label: file.name, value: file.name }))
)
const sortedRules = computed(() =>
  (ruleSet.value?.rules ?? [])
    .map((rule, index) => ({ rule, index }))
    .sort((a, b) => (a.rule.priority === b.rule.priority ? a.index - b.index : a.rule.priority - b.rule.priority))
)

const protocolOptions = [
  { label: '全部协议', value: 'all' },
  { label: 'SOCKS5', value: 'socks5' },
  { label: 'HTTP', value: 'http' }
]

const selectedProtocol = computed({
  get: () => draft.value.protocols[0] ?? 'all',
  set: (val: string) => {
    draft.value.protocols = val ? [val] : ['all']
  }
})
const matchTypeOptions = [
  { label: '任意', value: 'any' },
  { label: '单个 IP', value: 'ip' },
  { label: 'CIDR', value: 'cidr' },
  { label: '域名', value: 'domain' },
  { label: '通配域名', value: 'wildcard' }
]
const outboundOptions = [
  { label: '直连', value: 'default' },
  { label: '网卡', value: 'interface' },
  { label: '拦截', value: 'intercept' }
]

function emptyRule(): RuleDraft {
  return {
    id: `rule-${Date.now()}`,
    name: '新规则',
    enabled: true,
    priority: 100,
    protocols: ['all'],
    matchType: 'domain',
    targets: [],
    targetText: '',
    outbound: {
      mode: 'default',
      localIp: '',
      interface: ''
    },
    remark: ''
  }
}

function cloneRule(rule: RouteRule): RuleDraft {
  return {
    ...JSON.parse(JSON.stringify(rule)),
    targetText: rule.targets.join('\n')
  }
}

function normalizeRule(rule: RuleDraft): RouteRule {
  let protocols = rule.protocols
  if (protocols.includes('all')) {
    protocols = ['socks5', 'http']
  } else if (protocols.length === 0) {
    protocols = ['socks5', 'http']
  }
  return {
    id: rule.id.trim(),
    name: rule.name.trim(),
    enabled: rule.enabled,
    priority: Number(rule.priority) || 100,
    protocols: protocols,
    matchType: rule.matchType,
    targets: rule.matchType === 'any' ? ['*'] : rule.targetText.split(/\r?\n|,/).map((item) => item.trim()).filter(Boolean),
    outbound: {
      mode: rule.outbound.mode,
      localIp: rule.outbound.localIp.trim(),
      interface: rule.outbound.interface.trim()
    },
    remark: rule.remark.trim()
  }
}

function outboundText(rule: RouteRule): string {
  if (rule.outbound.mode === 'interface') return rule.outbound.interface || '-'
  if (rule.outbound.mode === 'intercept') return '拦截'
  return '直连'
}

function matchTypeLabel(type: string): string {
  const map: Record<string, string> = { any: '任意', ip: 'IP', cidr: 'CIDR', domain: '域名', wildcard: '通配' }
  return map[type] || type
}

function targetPlaceholder(type: string): string {
  const map: Record<string, string> = {
    ip: '例如：192.168.1.1\n一行一个配置',
    cidr: '例如：192.168.1.0/24\n一行一个配置',
    domain: '例如：example.com\n一行一个配置',
    wildcard: '例如：*.example.com\n一行一个配置'
  }
  return map[type] || '每行一个目标地址'
}

function copyRuleId() {
  if (!draft.value.id) return
  window.navigator.clipboard.writeText(draft.value.id)
  message.success('已复制')
}

function formatTargets(rule: RouteRule): string {
  if (rule.matchType === 'any') return '*'
  return rule.targets.join(', ')
}

async function loadAll() {
  loading.value = true
  error.value = ''
  try {
    if (!config.draft) await config.load()
    files.value = await listRouteFiles()
    deleteTarget.value = deletableFileOptions.value[0]?.value ?? ''
    ruleSet.value = await loadRouteFile(activeFile.value)
    interfaces.value = await getNetworkInterfaces()
  } catch (err) {
    error.value = friendlyError(err)
  } finally {
    loading.value = false
  }
}

async function switchFile(name: string) {
  if (!name || name === activeFile.value) return
  await setActiveRouteFile(name)
  await config.load()
  await loadAll()
  message.success('规则文件已切换')
}

async function saveRules() {
  if (!ruleSet.value) return
  saving.value = true
  try {
    ruleSet.value.updatedAt = new Date().toISOString()
    await saveRouteFile(activeFile.value, ruleSet.value)
    files.value = await listRouteFiles()
    message.success('路由规则已保存')
  } catch (err) {
    message.error(friendlyError(err))
  } finally {
    saving.value = false
  }
}

async function saveRouteSwitch() {
  if (!config.draft) return
  await config.save(server.status.running)
  message.success(config.draft.route.enabled ? '路由规则已启用' : '路由规则已停用')
}

async function createFile() {
  const name = window.prompt('新规则文件名，例如 office.rule')
  if (!name) return
  try {
    await createRouteFile(name.trim())
    files.value = await listRouteFiles()
    message.success('规则文件已创建')
  } catch (err) {
    message.error(friendlyError(err))
  }
}

async function copyFile() {
  if (!ruleSet.value) return
  const name = window.prompt('另存为文件名，例如 backup.rule')
  if (!name) return
  try {
    await saveRouteFile(name.trim(), { ...ruleSet.value, name: name.trim().replace(/\.rule$/, '') })
    files.value = await listRouteFiles()
    message.success('规则文件已另存')
  } catch (err) {
    message.error(friendlyError(err))
  }
}

function confirmDeleteFile() {
  if (!deleteTarget.value) return
  dialog.warning({
    title: '删除规则文件',
    content: deleteTarget.value,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await deleteRouteFile(deleteTarget.value)
        await config.load()
        await loadAll()
        message.success('规则文件已删除')
      } catch (err) {
        message.error(friendlyError(err))
      }
    }
  })
}

function addRule() {
  editingIndex.value = -1
  draft.value = emptyRule()
  modalVisible.value = true
}

function editRule(index: number) {
  if (!ruleSet.value) return
  editingIndex.value = index
  draft.value = cloneRule(ruleSet.value.rules[index])
  modalVisible.value = true
}

function removeRule(index: number) {
	if (!ruleSet.value) return
	if (ruleSet.value.rules[index]?.id === 'default') {
		message.warning('默认兜底规则不能删除')
		return
	}
	ruleSet.value.rules.splice(index, 1)
}

function saveDraft() {
  if (!ruleSet.value) return
  const next = normalizeRule(draft.value)
  if (!next.id || !next.name || next.targets.length === 0) {
    message.error('请补全规则名称、ID 和目标')
    return
  }
  if (editingIndex.value >= 0) {
    ruleSet.value.rules.splice(editingIndex.value, 1, next)
  } else {
    ruleSet.value.rules.push(next)
  }
  modalVisible.value = false
}

onMounted(loadAll)
</script>

<template>
  <section class="route-page">
    <NSpin :show="loading">
      <div class="page-shell">
        <div class="page-header">
          <div class="page-header-main">
            <div class="page-header-icon">
              <NIcon :component="Route" />
            </div>
            <div>
              <h2 class="page-title">路由规则</h2>
              <p class="page-subtitle">配置流量匹配与出口策略</p>
            </div>
          </div>
          <div class="page-header-actions">
            <NSwitch v-if="config.draft" v-model:value="config.draft.route.enabled" @update:value="saveRouteSwitch">
              <template #checked>启用</template>
              <template #unchecked>停用</template>
            </NSwitch>
            <NButton type="primary" :loading="saving" @click="saveRules">
              <template #icon><NIcon :component="Save" /></template>
              保存
            </NButton>
          </div>
        </div>

        <div v-if="error" class="page-error">{{ error }}</div>

        <template v-if="ruleSet">
          <section class="config-card">
            <div class="file-toolbar">
              <div class="file-toolbar-left">
                <NSelect :value="activeFile" :options="fileOptions" class="file-select" @update:value="switchFile" />
                <NTooltip placement="bottom">
                  <template #trigger>
                    <button class="info-btn" type="button">
                      <NIcon :component="Info" />
                    </button>
                  </template>
                  <div class="info-tooltip">
                    <div v-if="ruleSet.name" class="info-row"><span class="info-label">名称</span><span class="info-val">{{ ruleSet.name }}</span></div>
                    <div v-if="ruleSet.description" class="info-row"><span class="info-label">描述</span><span class="info-val">{{ ruleSet.description }}</span></div>
                    <div class="info-row"><span class="info-label">版本</span><span class="info-val">v{{ ruleSet.version }}</span></div>
                    <div v-if="ruleSet.updatedAt" class="info-row"><span class="info-label">更新</span><span class="info-val">{{ ruleSet.updatedAt }}</span></div>
                  </div>
                </NTooltip>
              </div>
              <div class="file-toolbar-right">
                <NButton secondary size="small" @click="createFile">
                  <template #icon><NIcon :component="FilePlus2" /></template>
                  新建文件
                </NButton>
                <NButton secondary size="small" @click="copyFile">
                  <template #icon><NIcon :component="Copy" /></template>
                  另存为
                </NButton>
                <div class="delete-action">
                  <NSelect v-model:value="deleteTarget" :options="deletableFileOptions" size="small" class="delete-select" placeholder="选择文件" />
                  <NButton secondary size="small" type="error" :disabled="!deleteTarget" @click="confirmDeleteFile">
                    <template #icon><NIcon :component="Trash2" /></template>
                    删除
                  </NButton>
                </div>
              </div>
            </div>
          </section>

          <section class="config-card config-card-spaced">
            <div class="card-head">
              <div class="card-title-wrap">
                <div class="card-icon">
                  <NIcon :component="Route" />
                </div>
                <div>
                  <div class="card-title">规则列表</div>
                  <div class="card-subtitle">共 {{ sortedRules.length }} 条</div>
                </div>
              </div>
              <NButton type="primary" size="small" @click="addRule">
                <template #icon><NIcon :component="Plus" /></template>
                添加规则
              </NButton>
            </div>

            <div class="table-wrap">
              <table v-if="sortedRules.length > 0" class="rule-table">
                <thead>
                  <tr>
                    <th class="col-status">状态</th>
                    <th class="col-pri">优先级</th>
                    <th class="col-name">名称</th>
                    <th class="col-proto">协议</th>
                    <th class="col-match">匹配</th>
                    <th class="col-target">目标</th>
                    <th class="col-outbound">出口</th>
                    <th class="col-actions">操作</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="{ rule, index } in sortedRules" :key="rule.id" :class="{ 'row-disabled': !rule.enabled }">
                    <td>
                      <NSwitch
                        :value="rule.enabled"
                        size="small"
                        @update:value="(val: boolean) => { if (ruleSet) ruleSet.rules[index].enabled = val }"
                      />
                    </td>
                    <td><span class="pri-badge">P{{ rule.priority }}</span></td>
                    <td class="cell-name">{{ rule.name }}</td>
                    <td>
                      <div class="proto-tags">
                        <span v-if="rule.protocols.includes('socks5')" class="proto-tag tag-s5">S5</span>
                        <span v-if="rule.protocols.includes('http')" class="proto-tag tag-http">HTTP</span>
                      </div>
                    </td>
                    <td><span class="match-tag">{{ matchTypeLabel(rule.matchType) }}</span></td>
                    <td class="cell-target" :title="formatTargets(rule)">{{ formatTargets(rule) }}</td>
                    <td>{{ outboundText(rule) }}</td>
                    <td>
                      <div class="row-actions">
                        <button class="row-btn" type="button" @click="editRule(index)">
                          <NIcon :component="Pencil" />
                        </button>
                        <button
                          class="row-btn danger"
                          type="button"
                          :disabled="rule.id === 'default'"
                          @click="removeRule(index)"
                        >
                          <NIcon :component="Trash2" />
                        </button>
                      </div>
                    </td>
                  </tr>
                </tbody>
              </table>

              <div v-else class="rules-empty">
                <div class="rules-empty-icon">
                  <NIcon :component="Route" />
                </div>
                <p class="rules-empty-text">暂无路由规则</p>
                <p class="rules-empty-hint">点击上方「添加规则」创建第一条规则</p>
              </div>
            </div>
          </section>
        </template>
      </div>
    </NSpin>

    <NDrawer v-model:show="modalVisible" :width="'45%'" placement="left" class="route-drawer">
      <NDrawerContent :closable="true" :header-style="{ padding: 0 }" :body-content-style="{ padding: 0 }" :footer-style="{ padding: '14px 20px' }">
        <template #header>
          <div class="dlg-header">
            <div class="dlg-icon"><NIcon :component="Plus" :size="16" /></div>
            <div>
              <div class="dlg-title">{{ editingIndex >= 0 ? '编辑规则' : '新增规则' }}</div>
              <div class="dlg-desc">配置代理规则以控制流量转发</div>
            </div>
          </div>
        </template>

        <div class="dlg-body">
          <div class="dlg-section">
            <div class="dlg-section-head">
              <span class="dlg-bar dlg-bar--green"></span>
              <span class="dlg-section-label">基本信息</span>
            </div>
            <div class="dlg-row">
              <label class="dlg-label">规则名称</label>
              <div class="dlg-ctrl"><NInput v-model:value="draft.name" placeholder="公司内网分流" size="small" /></div>
            </div>
            <div v-if="editingIndex >= 0" class="dlg-row">
              <label class="dlg-label">规则 ID <NTooltip trigger="hover"><template #trigger><span class="dlg-tip"><NIcon :component="Info" :size="13" /></span></template><span>唯一标识，创建后不可修改</span></NTooltip></label>
              <div class="dlg-ctrl">
                <code class="dlg-id-code">{{ draft.id }}</code>
              </div>
            </div>
            <div class="dlg-row">
              <label class="dlg-label">优先级 <NTooltip trigger="hover"><template #trigger><span class="dlg-tip"><NIcon :component="Info" :size="13" /></span></template><span>数值越小优先级越高<br /><b>例</b> 10 高优先 / 100 默认</span></NTooltip></label>
              <div class="dlg-ctrl"><NInputNumber v-model:value="draft.priority" :min="1" placeholder="100" size="small" class="dlg-input-sm" /></div>
            </div>
            <div class="dlg-row">
              <label class="dlg-label">启用 <NTooltip trigger="hover"><template #trigger><span class="dlg-tip"><NIcon :component="Info" :size="13" /></span></template><span>关闭后规则不参与匹配</span></NTooltip></label>
              <div class="dlg-ctrl">
                <div class="dlg-switch-row">
                  <NSwitch v-model:value="draft.enabled" size="small" />
                  <span class="dlg-switch-text" :class="{ active: draft.enabled }">{{ draft.enabled ? '已启用' : '已停用' }}</span>
                </div>
              </div>
            </div>
          </div>

          <div class="dlg-divider"></div>

          <div class="dlg-section">
            <div class="dlg-section-head">
              <span class="dlg-bar dlg-bar--cyan"></span>
              <span class="dlg-section-label">匹配条件</span>
            </div>
            <div class="dlg-row">
              <label class="dlg-label">协议 <NTooltip trigger="hover"><template #trigger><span class="dlg-tip"><NIcon :component="Info" :size="13" /></span></template><span>规则对哪些协议生效<br /><b>全部协议</b> 所有协议<br /><b>SOCKS5</b> SOCKS5 代理<br /><b>HTTP</b> CONNECT 代理</span></NTooltip></label>
              <div class="dlg-ctrl"><NSelect v-model:value="selectedProtocol" :options="protocolOptions" size="small" /></div>
            </div>
            <div class="dlg-row">
              <label class="dlg-label">匹配类型 <NTooltip trigger="hover"><template #trigger><span class="dlg-tip"><NIcon :component="Info" :size="13" /></span></template><span>如何匹配目标地址<br /><b>任意</b> 所有流量<br /><b>IP</b> 精确 IP<br /><b>CIDR</b> 网段<br /><b>域名</b> 完整域名<br /><b>通配</b> 如 *.example.com</span></NTooltip></label>
              <div class="dlg-ctrl"><NSelect v-model:value="draft.matchType" :options="matchTypeOptions" size="small" /></div>
            </div>
            <div v-if="draft.matchType !== 'any'" class="dlg-row dlg-row--top">
              <label class="dlg-label">目标地址 <NTooltip trigger="hover"><template #trigger><span class="dlg-tip"><NIcon :component="Info" :size="13" /></span></template><span>每行一个目标地址</span></NTooltip></label>
              <div class="dlg-ctrl dlg-ctrl--stack">
                <NInput v-model:value="draft.targetText" type="textarea" :autosize="{ minRows: 3, maxRows: 6 }" :placeholder="targetPlaceholder(draft.matchType)" />
                <span class="dlg-hint">每行一个</span>
              </div>
            </div>
          </div>

          <div class="dlg-divider"></div>

          <div class="dlg-section">
            <div class="dlg-section-head">
              <span class="dlg-bar dlg-bar--amber"></span>
              <span class="dlg-section-label">出口设置</span>
            </div>
            <div class="dlg-row">
              <label class="dlg-label">出口模式 <NTooltip trigger="hover"><template #trigger><span class="dlg-tip"><NIcon :component="Info" :size="13" /></span></template><span>匹配流量的出口方式<br /><b>直连</b> 系统路由<br /><b>网卡</b> 绑定网卡<br /><b>拦截</b> 拒绝连接</span></NTooltip></label>
              <div class="dlg-ctrl"><NSelect v-model:value="draft.outbound.mode" :options="outboundOptions" size="small" /></div>
            </div>
            <div v-if="draft.outbound.mode === 'interface'" class="dlg-row">
              <label class="dlg-label">网卡 <NTooltip trigger="hover"><template #trigger><span class="dlg-tip"><NIcon :component="Info" :size="13" /></span></template><span>流量从指定网卡发出</span></NTooltip></label>
              <div class="dlg-ctrl"><NSelect v-model:value="draft.outbound.interface" :options="interfaceOptions" filterable placeholder="选择网卡" size="small" /></div>
            </div>
          </div>

          <div class="dlg-divider"></div>

          <div class="dlg-section">
            <div class="dlg-section-head">
              <span class="dlg-bar dlg-bar--gray"></span>
              <span class="dlg-section-label">备注</span>
            </div>
            <div class="dlg-row dlg-row--top">
              <label class="dlg-label">备注 <NTooltip trigger="hover"><template #trigger><span class="dlg-tip"><NIcon :component="Info" :size="13" /></span></template><span>可选，记录规则用途</span></NTooltip></label>
              <div class="dlg-ctrl">
                <NInput v-model:value="draft.remark" type="textarea" :autosize="{ minRows: 2, maxRows: 4 }" placeholder="可选" />
              </div>
            </div>
          </div>
        </div>

        <template #footer>
          <NSpace justify="end" :size="8" style="width: 100%">
            <NButton @click="modalVisible = false">取消</NButton>
            <NButton type="primary" @click="saveDraft">保存规则</NButton>
          </NSpace>
        </template>
      </NDrawerContent>
    </NDrawer>
  </section>
</template>

<style scoped>
.route-page {
  width: 100%;
}

.page-shell {
  max-width: 1080px;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.page-header-main {
  display: flex;
  align-items: center;
  gap: 14px;
}

.page-header-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.page-header-icon {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: color-mix(in srgb, var(--panel) 86%, var(--fg-soft) 14%);
  color: var(--fg);
  font-size: 18px;
  border: 1px solid var(--border);
}

.page-title {
  margin: 0;
  font-size: 28px;
  line-height: 1.15;
  font-weight: 600;
  color: var(--fg);
}

.page-subtitle {
  margin: 6px 0 0;
  font-size: 14px;
  color: var(--fg-soft);
}

.page-error {
  padding: 12px 16px;
  color: #f87171;
  background: rgba(239, 68, 68, 0.08);
  border: 1px solid rgba(239, 68, 68, 0.2);
  border-radius: 10px;
  font-size: 13px;
}

.config-card {
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: 12px;
  box-shadow: 0 10px 30px rgba(15, 23, 42, 0.04);
  overflow: hidden;
}

.config-card-spaced {
  margin-top: 0;
}

.card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 20px 24px 16px;
}

.card-title-wrap {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.card-icon {
  width: 36px;
  height: 36px;
  border-radius: 10px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
  color: #7c3aed;
  background: rgba(139, 92, 246, 0.12);
}

.card-title {
  font-size: 16px;
  line-height: 1;
  font-weight: 600;
  color: var(--fg);
}

.card-subtitle {
  margin-top: 5px;
  font-size: 13px;
  color: var(--fg-soft);
}

/* ---- file toolbar ---- */

.file-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 14px 20px;
}

.file-toolbar-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.file-toolbar-right {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.file-select {
  width: 220px;
}

.info-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  color: var(--muted);
  cursor: pointer;
  background: transparent;
  border: 1px solid transparent;
  border-radius: 6px;
  font-size: 15px;
  transition: all 0.15s;
}

.info-btn:hover {
  color: var(--text);
  background: var(--bg3);
  border-color: var(--border);
}

.info-tooltip {
  display: flex;
  flex-direction: column;
  gap: 6px;
  font-size: 13px;
  line-height: 1.5;
}

.info-row {
  display: flex;
  gap: 8px;
}

.info-label {
  color: var(--muted);
  min-width: 36px;
}

.info-val {
  color: var(--text);
  font-family: var(--mono);
  font-size: 12px;
}

.delete-action {
  display: flex;
  align-items: center;
  gap: 6px;
}

.delete-select {
  width: 140px;
}

/* ---- rule table ---- */

.table-wrap {
  padding: 0 16px 16px;
  overflow-x: auto;
}

.rule-table {
  width: 100%;
  border-collapse: collapse;
  table-layout: fixed;
  font-size: 13px;
}

.rule-table th {
  padding: 8px 12px;
  color: var(--muted);
  font-size: 11px;
  font-weight: 500;
  letter-spacing: 0.04em;
  text-align: left;
  text-transform: uppercase;
  border-bottom: 1px solid var(--border);
  white-space: nowrap;
}

.rule-table td {
  padding: 10px 12px;
  color: var(--text);
  border-bottom: 1px solid var(--line-soft);
  vertical-align: middle;
}

.rule-table tr:last-child td {
  border-bottom: 0;
}

.rule-table tr.row-disabled {
  opacity: 0.5;
}

.col-status { width: 56px; }
.col-pri { width: 72px; }
.col-name { width: auto; }
.col-proto { width: 90px; }
.col-match { width: 72px; }
.col-target { width: auto; }
.col-outbound { width: 88px; }
.col-actions { width: 76px; }

.pri-badge {
  display: inline-flex;
  align-items: center;
  padding: 2px 7px;
  font-family: var(--mono);
  font-size: 11px;
  font-weight: 600;
  color: var(--accent3);
  background: rgba(245, 158, 11, 0.12);
  border: 1px solid rgba(245, 158, 11, 0.2);
  border-radius: 4px;
}

.cell-name {
  font-weight: 500;
}

.proto-tags {
  display: flex;
  gap: 4px;
}

.proto-tag {
  display: inline-flex;
  align-items: center;
  padding: 2px 6px;
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.03em;
  border-radius: 4px;
}

.proto-tag.tag-s5 {
  color: #60a5fa;
  background: rgba(59, 130, 246, 0.14);
}

.proto-tag.tag-http {
  color: #fbbf24;
  background: rgba(245, 158, 11, 0.14);
}

.match-tag {
  display: inline-flex;
  align-items: center;
  padding: 2px 8px;
  font-size: 11px;
  font-weight: 600;
  color: #a78bfa;
  background: rgba(139, 92, 246, 0.12);
  border-radius: 4px;
}

.cell-target {
  font-family: var(--mono);
  font-size: 11.5px;
  max-width: 240px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.row-actions {
  display: flex;
  align-items: center;
  gap: 2px;
}

.row-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  color: var(--muted);
  cursor: pointer;
  background: transparent;
  border: 1px solid transparent;
  border-radius: 6px;
  font-size: 14px;
  transition: all 0.15s;
}

.row-btn:hover {
  color: var(--text);
  background: var(--bg3);
  border-color: var(--border);
}

.row-btn.danger:hover {
  color: var(--accent4);
  background: rgba(239, 68, 68, 0.08);
  border-color: rgba(239, 68, 68, 0.2);
}

.row-btn:disabled {
  opacity: 0.3;
  cursor: not-allowed;
}

/* ---- empty state ---- */

.rules-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 48px 16px;
}

.rules-empty-icon {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: var(--subtle);
  background: var(--bg3);
  border: 1px solid var(--border);
  font-size: 22px;
  margin-bottom: 12px;
}

.rules-empty-text {
  margin: 0;
  font-size: 14px;
  font-weight: 500;
  color: var(--fg-soft);
}

.rules-empty-hint {
  margin: 4px 0 0;
  font-size: 12px;
  color: var(--subtle);
}

/* ---- drawer ---- */

.route-drawer :deep(.n-drawer-body-content-wrapper) {
  padding: 0;
  height: 100%;
}

.dlg-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 16px 20px;
}

.dlg-icon {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  background: rgba(16, 185, 129, 0.1);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #10b981;
  flex-shrink: 0;
}

.dlg-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--fg);
  line-height: 1.3;
}

.dlg-desc {
  font-size: 12px;
  color: var(--muted);
  margin-top: 1px;
}

.dlg-body {
  padding: 0 16px 20px;
  overflow-y: auto;
  flex: 1;
}

.dlg-body::-webkit-scrollbar {
  width: 4px;
}

.dlg-body::-webkit-scrollbar-track {
  background: transparent;
}

.dlg-body::-webkit-scrollbar-thumb {
  background: var(--border);
  border-radius: 2px;
}

.dlg-section {
  margin-bottom: 20px;
}

.dlg-section:last-child {
  margin-bottom: 0;
}

.dlg-section-head {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 14px;
}

.dlg-bar {
  width: 3px;
  height: 14px;
  border-radius: 2px;
  flex-shrink: 0;
}

.dlg-bar--green { background: #10b981; }
.dlg-bar--cyan { background: #06b6d4; }
.dlg-bar--amber { background: #f59e0b; }
.dlg-bar--gray { background: #a1a1aa; }

.dlg-section-label {
  font-size: 13px;
  font-weight: 500;
  color: var(--fg);
}

.dlg-divider {
  height: 1px;
  background: var(--border);
  margin-bottom: 20px;
}

.dlg-row {
  display: flex;
  flex-direction: column;
  gap: 5px;
  margin-bottom: 12px;
}

.dlg-row:last-child {
  margin-bottom: 0;
}

.dlg-label {
  font-size: 12px;
  color: var(--muted);
  line-height: 1;
  display: inline-flex;
  align-items: center;
  gap: 3px;
}

.dlg-tip {
  display: inline-flex;
  align-items: center;
  color: var(--subtle);
  cursor: help;
  transition: color 0.15s;
}

.dlg-tip:hover {
  color: var(--accent);
}

.dlg-ctrl {
  min-width: 0;
}

.dlg-ctrl--stack {
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.dlg-hint {
  font-size: 11px;
  color: var(--subtle);
}

.dlg-id-row {
  display: flex;
  align-items: center;
  gap: 6px;
}

.dlg-id-code {
  font-family: var(--mono);
  font-size: 12px;
  color: var(--muted);
  background: color-mix(in srgb, var(--panel) 80%, var(--fg-soft) 20%);
  padding: 5px 10px;
  border-radius: 6px;
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  user-select: all;
}

.dlg-copy-btn {
  width: 26px;
  height: 26px;
  border-radius: 6px;
  border: 1px solid var(--border);
  background: transparent;
  color: var(--muted);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  transition: all 0.15s;
}

.dlg-copy-btn:hover {
  background: var(--bg3);
  color: var(--fg);
}

.dlg-input-sm {
  width: 100%;
}

.dlg-switch-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.dlg-switch-text {
  font-size: 13px;
  color: var(--muted);
  transition: color 0.15s;
}

.dlg-switch-text.active {
  color: #10b981;
}

.route-page :deep(.n-input),
.route-page :deep(.n-base-selection),
.route-page :deep(.n-input-number) {
  --n-border-radius: 8px !important;
}

.route-page :deep(.n-input .n-input__border),
.route-page :deep(.n-base-selection .n-base-selection__border),
.route-page :deep(.n-input-number .n-input-wrapper) {
  border-color: var(--border) !important;
}

.route-page :deep(.n-input:hover .n-input__border),
.route-page :deep(.n-base-selection:hover .n-base-selection__border),
.route-page :deep(.n-input-number:hover .n-input-wrapper) {
  border-color: color-mix(in srgb, var(--fg) 22%, var(--border) 78%) !important;
}

@media (max-width: 720px) {
  .page-header,
  .page-header-main {
    flex-direction: column;
    align-items: flex-start;
  }

  .page-header-actions {
    width: 100%;
    justify-content: flex-end;
  }

  .file-toolbar {
    flex-direction: column;
    align-items: flex-start;
  }

  .file-toolbar-right {
    width: 100%;
  }

  .file-select {
    width: 100%;
  }

  .dlg-row {
    gap: 4px;
  }
}
</style>
