<template>
  <div class="keys-container">
    <div class="header-section">
      <h2>网关密钥管理 (Gateway Keys)</h2>
      <button @click="showAddModal = true" class="btn-primary">生成新密钥</button>
    </div>

    <div class="stats-grid">
      <div class="stat-card">
        <span class="label">活跃密钥</span>
        <span class="value">{{ keys.length }}</span>
      </div>
    </div>

    <div class="table-card">
      <table class="data-table">
        <thead>
          <tr>
            <th>ID</th>
            <th>密钥内容 (Key Value)</th>
            <th>今日额度 (Daily Limit)</th>
            <th>已消耗 (Used)</th>
            <th>状态</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="key in keys" :key="key.id">
            <td>{{ key.id }}</td>
            <td class="key-cell">
              <code>{{ key.key_value }}</code>
              <button @click="copyKey(key.key_value)" class="btn-icon">📋</button>
            </td>
            <td>{{ key.daily_limit === -1 ? '无限' : key.daily_limit }}</td>
            <td>{{ key.used_tokens }}</td>
            <td>
              <span class="tag" :class="key.used_tokens >= key.daily_limit && key.daily_limit !== -1 ? 'tag-error' : 'tag-success'">
                {{ key.used_tokens >= key.daily_limit && key.daily_limit !== -1 ? '额度耗尽' : '正常' }}
              </span>
            </td>
            <td>
              <button @click="deleteKey(key.id)" class="btn-text-danger">删除</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-if="showAddModal" class="modal-overlay">
      <div class="modal-content">
        <h3>创建新网关密钥</h3>
        <div class="form-group">
          <label>自定义密钥名称 (可选)</label>
          <input v-model="newKey.key_value" placeholder="留空则随机生成" />
        </div>
        <div class="form-group">
          <label>每日额度 (Daily Limit)</label>
          <input type="number" v-model.number="newKey.daily_limit" placeholder="-1 代表无限" />
        </div>
        <div class="modal-actions">
          <button @click="showAddModal = false" class="btn-secondary">取消</button>
          <button @click="handleAdd" class="btn-primary">确认创建</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { GatewayKeysAPI } from '../api'

const keys = ref([])
const showAddModal = ref(false)
const newKey = ref({ key_value: '', daily_limit: -1 })

const fetchKeys = async () => {
  try {
    const data = await GatewayKeysAPI.list()
    keys.ref = data
  } catch (err) {
    console.error('加载失败')
  }
}

const handleAdd = async () => {
  try {
    await GatewayKeysAPI.create(newKey.value)
    showAddModal.value = false
    newKey.value = { key_value: '', daily_limit: -1 }
    await fetchKeys()
  } catch (err) {
    alert('创建失败')
  }
}

const deleteKey = async (id) => {
  if (!confirm('确定要撤销此密钥吗？所有关联的 Agent 将立即失效。')) return
  try {
    await GatewayKeysAPI.delete(id)
    await fetchKeys()
  } catch (err) {
    alert('删除失败')
  }
}

const copyKey = (val) => {
  navigator.clipboard.writeText(val)
  alert('密钥已复制到剪贴板')
}

onMounted(fetchKeys)
</script>

<style scoped>
.keys-container {
  padding: 20px;
}
.header-section {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}
.key-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}
code {
  background: #f4f4f4;
  padding: 2px 6px;
  border-radius: 4px;
  font-family: monospace;
}
.tag-success { background: #e6fffa; color: #2c7a7b; }
.tag-error { background: #fff5f5; color: #c53030; }
.modal-overlay {
  position: fixed;
  top: 0; left: 0; width: 100%; height: 100%;
  background: rgba(0,0,0,0.5);
  display: flex; align-items: center; justify-content: center;
}
.modal-content {
  background: white; padding: 24px; border-radius: 8px; width: 400px;
}
.form-group { margin-bottom: 16px; }
.form-group label { display: block; margin-bottom: 8px; }
.form-group input { width: 100%; padding: 8px; border: 1px solid #ddd; border-radius: 4px; }
.modal-actions { display: flex; justify-content: flex-end; gap: 12px; }
</style>