<script setup lang="ts">
import { computed, ref } from 'vue';
import { $t } from '@/locales';
import SvgIcon from '@/components/custom/svg-icon.vue';
import TableSiderLayout from '@/components/advanced/table-sider-layout.vue';
import AiKeyListPanel from './modules/ai-key-list-panel.vue';
import KeyScenarioPanel from './modules/key-scenario-panel.vue';

defineOptions({
  name: 'AiKeyList'
});

// 页面左右布局：左侧菜单切换 密钥列表/场景管理(场景是场景 Key 的分类字典，与密钥同域维护)
type PanelKey = 'keys' | 'scenario';

interface PanelMenuItem {
  key: PanelKey;
  label: string;
  desc: string;
  icon: string;
}

const activePanel = ref<PanelKey>('keys');

const menuItems = computed<PanelMenuItem[]>(() => [
  {
    key: 'keys',
    label: $t('page.gateway.aiKey.tabKeys'),
    desc: $t('page.gateway.aiKey.tabKeysDesc'),
    icon: 'lucide:key-round'
  },
  {
    key: 'scenario',
    label: $t('page.gateway.aiKey.tabScenario'),
    desc: $t('page.gateway.aiKey.tabScenarioDesc'),
    icon: 'lucide:shapes'
  }
]);

// 场景增/改/删后联动刷新密钥列表的场景列
const keyListRef = ref<InstanceType<typeof AiKeyListPanel>>();

function handleScenarioChanged() {
  keyListRef.value?.refresh();
}
</script>

<template>
  <TableSiderLayout :sider-title="$t('page.gateway.aiKey.title')">
    <template #sider>
      <div class="flex flex-col gap-4px">
        <div
          v-for="item in menuItems"
          :key="item.key"
          class="menu-item"
          :class="{ 'is-active': activePanel === item.key }"
          @click="activePanel = item.key"
        >
          <SvgIcon :icon="item.icon" class="h-24px w-24px shrink-0" :class="activePanel === item.key ? 'color-primary' : 'text-icon'" />
          <div class="min-w-0 flex-1">
            <div class="text-14px font-500">{{ item.label }}</div>
            <div class="truncate text-12px text-slate-400">{{ item.desc }}</div>
          </div>
        </div>
      </div>
    </template>
    <div class="h-full flex-col-stretch overflow-hidden">
      <AiKeyListPanel v-show="activePanel === 'keys'" ref="keyListRef" />
      <KeyScenarioPanel v-show="activePanel === 'scenario'" @changed="handleScenarioChanged" />
    </div>
  </TableSiderLayout>
</template>

<style scoped>
.menu-item {
  display: flex;
  cursor: pointer;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border: 1px solid transparent;
  border-radius: 8px;
  transition:
    background-color 0.2s,
    border-color 0.2s;
}

.menu-item:hover {
  background-color: rgb(var(--primary-color) / 0.05);
}

.menu-item.is-active {
  border-color: rgb(var(--primary-color) / 0.55);
  background-color: rgb(var(--primary-color) / 0.08);
}
</style>
