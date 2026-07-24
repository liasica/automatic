# 添加记录弹窗时间模式选择实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 「添加打卡记录」弹窗支持「随机 / 正常」两种补卡时间模式，正常模式固定预填上班 09:00、下班 18:00。

**Architecture:** 纯前端改动，只改 `AddRecordModal.vue`。新增 `timeMode` 状态与显式切换函数 `setTimeMode`；`prefillTimes` 在正常模式下直接填固定值、跳过历史查询。提交链路（`buildDateTime` 拼随机秒 → POST `/api/attendance/records`）不变。

**Tech Stack:** Vue 3 `<script setup>` + TypeScript + Tailwind CSS。项目无前端单测框架，验证方式为 `vue-tsc` type-check、eslint、浏览器手动操作。

**Spec:** `.superpowers/specs/2026-07-24-add-record-time-mode-design.md`

---

### Task 1: 实现时间模式选择（状态 + 预填逻辑 + 分段控件 UI）

**Files:**
- Modify: `dashboard/src/components/AddRecordModal.vue`

- [ ] **Step 1: 新增模式状态与常量**

在 `const checkInTime = ref('09:00')` 声明块之前（约第 22-25 行区域）加入：

```ts
// 补卡时间模式：random 按最近七天历史随机，normal 固定正常上下班时间
const timeMode = ref<'random' | 'normal'>('random')

// 正常上下班时间
const NORMAL_CHECK_IN = '09:00'
const NORMAL_CHECK_OUT = '18:00'

const timeModes = [
  { value: 'random', label: '随机' },
  { value: 'normal', label: '正常' },
] as const
```

- [ ] **Step 2: 弹窗打开时重置为随机模式**

修改 `watch(() => props.show, ...)`（第 44-52 行），在 `addCheckOut.value = true` 之后加一行：

```ts
watch(() => props.show, val => {
  if (val) {
    selectedUser.value = props.defaultUserId
    addCheckIn.value = true
    addCheckOut.value = true
    timeMode.value = 'random'
    const n = new Date()
    pickedDate.value = `${n.getFullYear()}-${pad(n.getMonth() + 1)}-${pad(n.getDate())}`
  }
})
```

直接赋值而非经切换函数，避免重开弹窗时与 `prefillKey` watch 重复触发预填。

- [ ] **Step 3: 新增显式切换函数**

在 `prefillTimes` 函数之后加入：

```ts
// 切换补卡时间模式：覆盖已填时间（含手动修改过的）
function setTimeMode(mode: 'random' | 'normal') {
  if (timeMode.value === mode) return
  timeMode.value = mode
  if (mode === 'normal') {
    // 使进行中的随机预填请求失效，避免异步结果覆盖固定值
    prefillToken++
    checkInDirty.value = false
    checkOutDirty.value = false
    checkInTime.value = NORMAL_CHECK_IN
    checkOutTime.value = NORMAL_CHECK_OUT
  }
  else {
    prefillTimes()
  }
}
```

- [ ] **Step 4: `prefillTimes` 感知模式**

在 `prefillTimes` 内（第 64 行起），`checkOutDirty.value = false` 之后、配置兜底赋值之前插入正常模式短路：

```ts
async function prefillTimes() {
  const u = currentUser.value
  if (!u) return
  const token = ++prefillToken
  checkInDirty.value = false
  checkOutDirty.value = false
  // 正常模式固定预填，不查询历史
  if (timeMode.value === 'normal') {
    checkInTime.value = NORMAL_CHECK_IN
    checkOutTime.value = NORMAL_CHECK_OUT
    return
  }
  checkInTime.value = u.check_in_latest || '09:00'
  checkOutTime.value = u.check_out_earliest || '18:00'
  // …… 以下现有历史查询与随机逻辑不变
}
```

`token` 变量在短路分支中未使用会触发 lint —— 短路 `return` 放在 `token` 声明之前也可以，但需保证切到正常模式后仍能让旧请求失效；保持上面顺序（先 `++prefillToken` 再短路）即可，`token` 未使用时改为不接收返回值：

```ts
async function prefillTimes() {
  const u = currentUser.value
  if (!u) return
  checkInDirty.value = false
  checkOutDirty.value = false
  // 正常模式固定预填，不查询历史
  if (timeMode.value === 'normal') {
    prefillToken++
    checkInTime.value = NORMAL_CHECK_IN
    checkOutTime.value = NORMAL_CHECK_OUT
    return
  }
  const token = ++prefillToken
  checkInTime.value = u.check_in_latest || '09:00'
  checkOutTime.value = u.check_out_earliest || '18:00'
  // …… 以下现有逻辑不变
}
```

采用第二种写法（`const token` 移到短路之后），无未使用变量问题。

- [ ] **Step 5: 「补卡时间」标签行加分段控件**

模板第 257 行的标签：

```html
<label class="mb-2 block font-mono text-xs uppercase tracking-wider text-muted">补卡时间</label>
```

替换为标签 + 分段控件同行布局：

```html
<div class="mb-2 flex items-center justify-between">
  <label class="font-mono text-xs uppercase tracking-wider text-muted">补卡时间</label>
  <div class="flex overflow-hidden rounded border border-oat">
    <button
      v-for="m in timeModes"
      :key="m.value"
      class="px-2.5 py-1 text-xs transition-colors"
      :class="timeMode === m.value ? 'bg-off-black text-white' : 'bg-white text-muted hover:text-off-black'"
      @click="setTimeMode(m.value)"
    >
      {{ m.label }}
    </button>
  </div>
</div>
```

- [ ] **Step 6: type-check 与 lint**

```bash
cd dashboard && npm run type-check && npx eslint src/components/AddRecordModal.vue
```

Expected: 均无报错、无 issue。

- [ ] **Step 7: Commit**

```bash
git add dashboard/src/components/AddRecordModal.vue
git commit -m "feat(dashboard): 添加记录弹窗支持正常/随机时间模式"
```

### Task 2: 浏览器验证

**Files:** 无代码改动。

- [ ] **Step 1: 启动 dev server 并打开弹窗**

启动 `dashboard` 的 vite dev server（`npm run dev`，需后端 API 可用时用 `make run` 或已部署实例；仅验证前端交互时 dev server 即可，历史查询失败会走配置兜底不影响模式切换验证）。

- [ ] **Step 2: 验证默认随机模式**

打开「添加记录」弹窗 → 分段控件默认选中「随机」，时间为随机预填值（或配置兜底值）。

- [ ] **Step 3: 验证切换行为**

- 点「正常」→ 上班变 `09:00`、下班变 `18:00`
- 手动改上班为 `09:30` → 点「随机」→ 两项被重新随机覆盖 → 再点「正常」→ 恢复 `09:00` / `18:00`
- 切到「正常」后切换用户 → 时间保持 `09:00` / `18:00`，无历史查询请求

- [ ] **Step 4: 验证弹窗重开重置**

选「正常」→ 关闭弹窗 → 重新打开 → 默认回到「随机」。
