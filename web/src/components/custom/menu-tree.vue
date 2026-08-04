<script setup lang="tsx">
import { computed, onMounted, ref, watch } from 'vue';
import type { TreeOption, TreeSelectInst } from 'naive-ui';
import { useBoolean } from '@sa/hooks';
import { fetchGetMenuTreeSelect } from '@/service/api/system';
import SvgIcon from '@/components/custom/svg-icon.vue';
import { $t } from '@/locales';
import { ALL_MODULES, MODULE_CONFIG, type RouteModule } from '@/constants/module';

defineOptions({
  name: 'MenuTree',
  inheritAttrs: false
});

interface Props {
  immediate?: boolean;
  showHeader?: boolean;
  /** 是否启用"默认路由"小房子(角色菜单授权用;C 菜单点击设为角色 DefaultRouter) */
  enableDefaultRouter?: boolean;
  [key: string]: any;
}

const props = withDefaults(defineProps<Props>(), {
  immediate: true,
  showHeader: true,
  enableDefaultRouter: false
});

const { bool: expandAll } = useBoolean();
const { bool: checkAll } = useBoolean();
// 公共页面分组节点 id(module 缺失的跨模块全局页如 home 归此);负值不与真实菜单雪花 id 冲突,
// 后端 saveRoleMenus 过滤 mid<=0 不入库。模块组 id=-(idx+1) 按 ALL_MODULES 顺序;公共组=-(length+1)。
const COMMON_GROUP_ID = -(ALL_MODULES.length + 1);
// 初始展开到分组层(根 0 + 各模块组 + 公共组),进来即可见各模块菜单
const DEFAULT_EXPANDED_KEYS: CommonType.IdType[] = [0, ...ALL_MODULES.map((_, i) => -(i + 1)), COMMON_GROUP_ID];
const expandedKeys = ref<CommonType.IdType[]>([...DEFAULT_EXPANDED_KEYS]);

const menuTreeRef = ref<TreeSelectInst | null>(null);
const checkedKeys = defineModel<CommonType.IdType[]>('checkedKeys', { required: false, default: [] });
const options = defineModel<Api.System.MenuList>('options', { required: false, default: [] });
const cascade = defineModel<boolean>('cascade', { required: false, default: true });
const loading = defineModel<boolean>('loading', { required: false, default: false });
const defaultRouter = defineModel<string>('defaultRouter', { required: false, default: '' });

// 按业务模块把菜单归类为各虚拟分组节点(角色授权树视觉分组用)。
// 分组节点用负 id(与真实菜单雪花 id 不冲突);后端 saveRoleMenus 已过滤 mid<=0,不会写入 sys_role_menu。
// 分组节点 menuType='M',不渲染默认路由小房子(renderHouse 仅认 C 菜单)。
function buildGroupedOptions(menus: Api.System.MenuList): Api.System.MenuList {
  const buckets = new Map<RouteModule, Api.System.MenuList>();
  ALL_MODULES.forEach(m => buckets.set(m, []));
  // module 缺失/非法(如 home 跨模块全局页)归「公共页面」组,不再兜底 admin 造成归属歧义
  const common: Api.System.MenuList = [];
  menus.forEach(m => {
    if (m.module && ALL_MODULES.includes(m.module as RouteModule)) {
      buckets.get(m.module as RouteModule)!.push(m);
    } else {
      common.push(m);
    }
  });

  const moduleGroups: Api.System.MenuList = ALL_MODULES.map((mod, idx) => ({
    id: -(idx + 1),
    label: $t(`module.${mod}` as App.I18n.I18nKey),
    icon: MODULE_CONFIG[mod].icon,
    menuType: 'M',
    module: mod,
    children: buckets.get(mod)!
  })) as Api.System.MenuList;

  // 公共页面组仅在存在空 module 菜单时出现,置顶显示
  if (!common.length) return moduleGroups;
  return [
    {
      id: COMMON_GROUP_ID,
      label: $t('module.common' as App.I18n.I18nKey),
      icon: 'mdi:earth',
      menuType: 'M',
      module: '',
      children: common
    },
    ...moduleGroups
  ] as Api.System.MenuList;
}

// 渲染树:把 options(平铺顶层菜单)包根 + 按模块分组。
// 必须放渲染层 computed:编辑角色时 immediate=false、getMenuList 不执行,options 由父组件直塞 data.menus;
// 若分组只在 getMenuList 里做,编辑场景会漏分组(平铺显示)。
const treeData = computed<Api.System.MenuList>(() => [
  {
    id: 0,
    label: $t('page.system.menu.rootName'),
    icon: 'material-symbols:home-outline-rounded',
    children: buildGroupedOptions(options.value ?? [])
  }
] as Api.System.MenuList);

async function getMenuList() {
  loading.value = true;
  const { error, data } = await fetchGetMenuTreeSelect();
  if (error) return;
  options.value = data;
  loading.value = false;
}

onMounted(() => {
  if (props.immediate) {
    getMenuList();
  }
});

watch(expandAll, newVal => {
  // 展开=全部节点;折叠=根 + 各分组层(仍可见各模块顶层菜单)
  expandedKeys.value = newVal ? getAllMenuIds(treeData.value) : [...DEFAULT_EXPANDED_KEYS];
});

// 由菜单 path 推导 routeKey(与后端 routeKey 规则一致:去首尾斜杠,/ 换 _)
function routeKeyOfPath(path?: string) {
  return path ? path.replace(/^\/+|\/+$/g, '').replace(/\//g, '_') : '';
}

// 默认路由小房子(C 菜单可打开页;点击设为角色 DefaultRouter)
function renderHouse(option: TreeOption) {
  if (!props.enableDefaultRouter || option.menuType !== 'C' || !option.path) return null;
  const key = routeKeyOfPath(String(option.path));
  const active = key === defaultRouter.value;
  return (
    <span
      class={`ml-6px inline-flex cursor-pointer text-16px ${active ? 'text-primary' : 'opacity-40 hover:opacity-80'}`}
      title={$t('page.system.role.setDefaultRouter')}
      onClick={(e: Event) => {
        e.stopPropagation();
        defaultRouter.value = key;
      }}
    >
      <SvgIcon icon={active ? 'mdi:home' : 'mdi:home-outline'} />
    </span>
  );
}

function renderLabel({ option }: { option: TreeOption }) {
  let label = option.label;
  if (label?.startsWith('route.') || label?.startsWith('menu.')) {
    label = $t(label as App.I18n.I18nKey);
  }
  const house = renderHouse(option);
  // 禁用的菜单显示红色
  if (option.status === '1') {
    return (
      <div class="flex items-center gap-4px text-error-200">
        {label}
        <SvgIcon icon="ri:prohibited-line" class="text-16px" />
        {house}
      </div>
    );
  }
  // 隐藏的菜单显示灰色
  if (option.visible === '1') {
    return (
      <div class="flex items-center gap-4px text-gray-400">
        {label}
        <SvgIcon icon="codex:hidden" class="text-21px" />
        {house}
      </div>
    );
  }
  return (
    <div class="flex items-center">
      {label}
      {house}
    </div>
  );
}

function renderPrefix({ option }: { option: TreeOption }) {
  const renderLocalIcon = String(option.icon).startsWith('local-icon-');
  let icon = renderLocalIcon ? undefined : String(option.icon ?? 'material-symbols:buttons-alt-outline-rounded');
  const localIcon = renderLocalIcon ? String(option.icon).replace('local-icon-', 'menu-') : undefined;
  if (icon === '#') {
    icon = 'material-symbols:buttons-alt-outline-rounded';
  }
  return <SvgIcon icon={icon} localIcon={localIcon} />;
}

function getAllMenuIds(menu: Api.System.MenuList) {
  const menuIds: CommonType.IdType[] = [];
  menu.forEach(item => {
    menuIds.push(item.id!);
    if (item.children) {
      menuIds.push(...getAllMenuIds(item.children));
    }
  });
  return menuIds;
}

/** 获取所有叶子节点的 ID（没有子节点的节点） */
function getLeafMenuIds(menu: Api.System.MenuList): CommonType.IdType[] {
  const leafIds: CommonType.IdType[] = [];
  menu.forEach(item => {
    if (!item.children || item.children.length === 0) {
      // 是叶子节点
      leafIds.push(item.id!);
    } else {
      // 有子节点，递归获取子节点中的叶子节点
      leafIds.push(...getLeafMenuIds(item.children));
    }
  });
  return leafIds;
}

function handleCheckedTreeNodeAll(checked: boolean) {
  if (checked) {
    checkedKeys.value = getAllMenuIds(treeData.value);
    return;
  }
  checkedKeys.value = [];
}

function getCheckedMenuIds(isCascade: boolean = false) {
  const menuIds = menuTreeRef.value?.getCheckedData()?.keys as string[];
  const indeterminateData = menuTreeRef.value?.getIndeterminateData();
  if (cascade.value || isCascade) {
    const parentIds: string[] = indeterminateData?.keys.filter(item => !menuIds?.includes(String(item))) as string[];
    menuIds?.push(...parentIds);
  }
  // 过滤虚拟分组节点(负 id)与根节点(0):后端 saveRoleMenus 亦过滤 mid<=0,此处双保险防污染 sys_role_menu
  return (menuIds ?? []).filter(id => Number(id) > 0);
}

watch(cascade, () => {
  if (cascade.value) {
    // 获取当前菜单树中的所有叶子节点ID
    const allLeafIds = getLeafMenuIds(treeData.value);
    // 筛选出当前选中项中的叶子节点
    const selectedLeafIds = checkedKeys.value.filter(id => allLeafIds.includes(id));
    // 重新设置选中状态为只包含叶子节点，让组件基于父子联动规则重新计算父节点状态
    checkedKeys.value = selectedLeafIds;
    return;
  }
  // 禁用父子联动时，将半选中的父节点也加入到选中列表
  checkedKeys.value = getCheckedMenuIds(true);
});

defineExpose({
  getCheckedMenuIds,
  refresh: getMenuList
});
</script>

<template>
  <div class="w-full flex-col gap-12px">
    <div v-if="showHeader" class="w-full flex-center">
      <NCheckbox v-model:checked="expandAll" :checked-value="true" :unchecked-value="false">展开/折叠</NCheckbox>
      <NCheckbox
        v-model:checked="checkAll"
        :checked-value="true"
        :unchecked-value="false"
        @update:checked="handleCheckedTreeNodeAll"
      >
        全选/反选
      </NCheckbox>
      <NCheckbox v-model:checked="cascade" :checked-value="true" :unchecked-value="false">父子联动</NCheckbox>
    </div>
    <NSpin class="resource h-full w-full py-6px pl-3px" content-class="h-full" :show="loading">
      <NTree
        ref="menuTreeRef"
        v-model:checked-keys="checkedKeys"
        v-model:expanded-keys="expandedKeys"
        multiple
        checkable
        :selectable="false"
        key-field="id"
        label-field="label"
        :data="treeData"
        :cascade="cascade"
        :loading="loading"
        virtual-scroll
        check-strategy="all"
        :render-label="renderLabel"
        :render-prefix="renderPrefix"
        v-bind="$attrs"
      />
    </NSpin>
  </div>
</template>

<style scoped lang="scss">
.resource {
  border-radius: 6px;
  border: 1px solid rgb(224, 224, 230);

  .n-tree {
    min-height: 200px;
    max-height: 300px;
    width: 100%;
    height: 100%;

    :deep(.n-tree__empty) {
      min-height: 200px;
      justify-content: center;
    }
  }

  .n-empty {
    justify-content: center;
  }
}
</style>
