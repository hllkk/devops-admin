<script setup lang="tsx">
import { onMounted } from 'vue';
import type { TreeOption } from 'naive-ui';
import { useLoading } from '@sa/hooks';
import { fetchGetMenuList } from '@/service/api/system';
import { handleTree } from '@/utils/common';
import SvgIcon from '@/components/custom/svg-icon.vue';
import { $t } from '@/locales';

defineOptions({
  name: 'MenuTreeSelect',
  inheritAttrs: false
});

interface Props {
  immediate?: boolean;
  [key: string]: any;
}

const props = withDefaults(defineProps<Props>(), {
  immediate: true
});

const value = defineModel<CommonType.IdType | null>('value', { required: false });
const options = defineModel<Api.System.MenuList>('options', { required: false, default: [] });

const { loading, startLoading, endLoading } = useLoading();

async function getMenuList() {
  startLoading();
  const { error, data } = await fetchGetMenuList();
  if (error) return;
  const { tree } = handleTree(data, { idField: 'menuId', filterFn: item => item.menuType !== 'F' });
  options.value = [
    {
      menuId: 0,
      // 根目录标签由 renderLabel 对 menuId=0 特判实时翻译，此处保留翻译值兜底
      menuName: $t('page.system.menu.rootName'),
      icon: 'material-symbols:home-outline-rounded',
      children: tree
    }
  ] as Api.System.MenuList;
  endLoading();
}

onMounted(() => {
  if (props.immediate) {
    getMenuList();
  }
});

function renderLabel({ option }: { option: TreeOption }) {
  // 根目录是前端拼装的虚拟节点（menuId=0），在渲染层实时翻译，确保跟随语言切换
  if (option.menuId === 0) {
    return <div>{$t('page.system.menu.rootName')}</div>;
  }
  let label = String(option.menuName);
  if (label?.startsWith('route.') || label?.startsWith('menu.')) {
    label = $t(label as App.I18n.I18nKey);
  }
  return <div>{label}</div>;
}

function renderPrefix({ option }: { option: TreeOption }) {
  const renderLocalIcon = String(option.icon).startsWith('local-icon-');
  const icon = renderLocalIcon ? undefined : String(option.icon);
  const localIcon = renderLocalIcon ? String(option.icon).replace('local-icon-', 'menu-') : undefined;
  return <SvgIcon icon={icon} localIcon={localIcon} />;
}
</script>

<template>
  <NTreeSelect
    v-model:value="value"
    filterable
    class="h-full"
    :loading="loading"
    key-field="menuId"
    label-field="menuName"
    :options="options"
    :default-expanded-keys="[0]"
    :render-tag="renderLabel"
    :render-label="renderLabel"
    :render-prefix="renderPrefix"
    v-bind="$attrs"
  />
</template>

<style scoped></style>
