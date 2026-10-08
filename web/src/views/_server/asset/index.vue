<script setup lang="ts">
import { computed, ref } from 'vue';
import { $t } from '@/locales';
import SvgIcon from '@/components/custom/svg-icon.vue';
import TableSiderLayout from '@/components/advanced/table-sider-layout.vue';
import AssetListPanel from './modules/asset-list-panel.vue';
import CredentialPanel from './modules/credential-panel.vue';

defineOptions({ name: 'ServerAsset' });

// 页面左右布局：左侧菜单切换 资产列表/凭据管理(凭据是资产的采集通道关联资源，同域维护)
type PanelKey = 'assets' | 'credential';

interface PanelMenuItem {
  key: PanelKey;
  label: string;
  desc: string;
  icon: string;
}

const activePanel = ref<PanelKey>('assets');

const menuItems = computed<PanelMenuItem[]>(() => [
  {
    key: 'assets',
    label: $t('page.server.asset.tabAssets'),
    desc: $t('page.server.asset.tabAssetsDesc'),
    icon: 'lucide:server'
  },
  {
    key: 'credential',
    label: $t('page.server.credential.tabCredential'),
    desc: $t('page.server.credential.tabCredentialDesc'),
    icon: 'lucide:key-round'
  }
]);

// 凭据增/删后资产表单的凭据下拉需刷新——监听凭据面板 changed 调资产面板 refresh
const assetListRef = ref<InstanceType<typeof AssetListPanel>>();

function handleCredentialChanged() {
  assetListRef.value?.refresh();
}
</script>

<template>
  <TableSiderLayout :sider-title="$t('page.server.asset.title')">
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
      <AssetListPanel v-show="activePanel === 'assets'" ref="assetListRef" />
      <CredentialPanel v-show="activePanel === 'credential'" @changed="handleCredentialChanged" />
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
