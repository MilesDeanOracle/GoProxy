<script setup lang="ts">
import { RotateCcw, Save } from 'lucide-vue-next'
import {
  NAlert,
  NButton,
  NForm,
  NFormItem,
  NGrid,
  NGridItem,
  NIcon,
  NInput,
  NInputNumber,
  NSelect,
  NSwitch,
  NSpin,
  useMessage
} from 'naive-ui'
import { useConfigStore } from '../stores/config'
import { useServerStore } from '../stores/server'

const config = useConfigStore()
const server = useServerStore()
const message = useMessage()

const logLevels = [
  { label: 'debug', value: 'debug' },
  { label: 'info', value: 'info' },
  { label: 'warn', value: 'warn' },
  { label: 'error', value: 'error' }
]

const logOutputs = [
  { label: '文件 + 控制台', value: 'both' },
  { label: '仅文件', value: 'file' },
  { label: '仅控制台', value: 'console' }
]

async function save() {
  await config.save(server.status.running)
  await server.refresh()
  message.success('配置已保存')
}
</script>

<template>
  <section class="config-page">
    <div class="section-actions">
      <div>
        <span class="section-kicker">CONFIG</span>
        <h2>服务参数</h2>
      </div>
      <div class="header-actions">
        <NButton :disabled="!config.dirty" secondary @click="config.reset">
          <template #icon>
            <NIcon :component="RotateCcw" />
          </template>
          重置
        </NButton>
        <NButton type="primary" :loading="config.saving" :disabled="!config.dirty" @click="save">
          <template #icon>
            <NIcon :component="Save" />
          </template>
          保存
        </NButton>
      </div>
    </div>

    <NAlert v-if="config.error" type="error" class="page-alert">
      {{ config.error }}
    </NAlert>
    <NAlert v-if="config.restartRequired" type="warning" class="page-alert">
      监听配置已保存，重启服务后生效。
    </NAlert>

    <NSpin :show="config.loading">
      <NForm v-if="config.draft" label-placement="top" class="config-form">
        <section class="panel form-panel">
          <div class="panel-head">
            <h3>入站协议</h3>
            <span class="tag">LISTENER</span>
          </div>
          <div class="form-body">
            <NGrid :cols="2" :x-gap="14" :y-gap="10" responsive="screen">
              <NGridItem>
                <NFormItem label="SOCKS5">
                  <NSwitch v-model:value="config.draft.server.socks5.enabled" />
                </NFormItem>
              </NGridItem>
              <NGridItem>
                <NFormItem label="HTTP CONNECT">
                  <NSwitch v-model:value="config.draft.server.http.enabled" />
                </NFormItem>
              </NGridItem>
              <NGridItem>
                <NFormItem label="SOCKS5 监听地址">
                  <NInput v-model:value="config.draft.server.socks5.host" />
                </NFormItem>
              </NGridItem>
              <NGridItem>
                <NFormItem label="HTTP 监听地址">
                  <NInput v-model:value="config.draft.server.http.host" />
                </NFormItem>
              </NGridItem>
              <NGridItem>
                <NFormItem label="SOCKS5 端口">
                  <NInputNumber v-model:value="config.draft.server.socks5.port" :min="1" :max="65535" />
                </NFormItem>
              </NGridItem>
              <NGridItem>
                <NFormItem label="HTTP 端口">
                  <NInputNumber v-model:value="config.draft.server.http.port" :min="1" :max="65535" />
                </NFormItem>
              </NGridItem>
            </NGrid>
          </div>
        </section>

        <section class="panel form-panel">
          <div class="panel-head">
            <h3>转发参数</h3>
            <span class="tag">RELAY</span>
          </div>
          <div class="form-body">
            <NGrid :cols="4" :x-gap="14" :y-gap="10" responsive="screen">
              <NGridItem>
                <NFormItem label="目标建连超时">
                  <NInputNumber v-model:value="config.draft.relay.dialTimeoutSec" :min="1" />
                </NFormItem>
              </NGridItem>
              <NGridItem>
                <NFormItem label="握手/写入超时">
                  <NInputNumber v-model:value="config.draft.relay.readTimeoutSec" :min="1" />
                </NFormItem>
              </NGridItem>
              <NGridItem>
                <NFormItem label="最大并发连接数">
                  <NInputNumber v-model:value="config.draft.relay.maxConnections" :min="1" />
                </NFormItem>
              </NGridItem>
              <NGridItem>
                <NFormItem label="Keep-Alive 间隔">
                  <NInputNumber v-model:value="config.draft.relay.keepaliveSec" :min="1" />
                </NFormItem>
              </NGridItem>
            </NGrid>
          </div>
        </section>

        <section class="panel form-panel">
          <div class="panel-head">
            <h3>日志</h3>
            <span class="tag">LOGGING</span>
          </div>
          <div class="form-body">
            <NGrid :cols="4" :x-gap="14" :y-gap="10" responsive="screen">
              <NGridItem>
                <NFormItem label="级别">
                  <NSelect v-model:value="config.draft.log.level" :options="logLevels" />
                </NFormItem>
              </NGridItem>
              <NGridItem>
                <NFormItem label="输出">
                  <NSelect v-model:value="config.draft.log.output" :options="logOutputs" />
                </NFormItem>
              </NGridItem>
              <NGridItem>
                <NFormItem label="单文件大小">
                  <NInputNumber v-model:value="config.draft.log.maxSizeMb" :min="1" />
                </NFormItem>
              </NGridItem>
              <NGridItem>
                <NFormItem label="备份数量">
                  <NInputNumber v-model:value="config.draft.log.maxBackups" :min="0" />
                </NFormItem>
              </NGridItem>
            </NGrid>
          </div>
        </section>
      </NForm>
    </NSpin>
  </section>
</template>
