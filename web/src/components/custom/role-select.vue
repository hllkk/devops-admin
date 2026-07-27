<script setup lang="ts">
import { ref } from 'vue';
import { useLoading } from '@sa/hooks';
import { fetchGetRoleSelect } from '@/service/api/system';

defineOptions({
  name: 'RoleSelect',
  inheritAttrs: false
});

interface Props {
  /** 是否多选(默认单选);单选 value=IdType,多选 value=IdType[] */
  multiple?: boolean;
}

withDefaults(defineProps<Props>(), { multiple: false });

const value = defineModel<CommonType.IdType | CommonType.IdType[] | null>('value', { required: false });

const { loading: roleLoading, startLoading: startRoleLoading, endLoading: endRoleLoading } = useLoading();

/** the enabled role options */
const roleOptions = ref<CommonType.Option<CommonType.IdType>[]>([]);

async function getRoleOptions() {
  startRoleLoading();
  const { error, data } = await fetchGetRoleSelect();

  if (!error) {
    roleOptions.value = data.map(item => ({
      label: item.roleName,
      value: item.roleId
    }));
  }
  endRoleLoading();
}

getRoleOptions();
</script>

<template>
  <NSelect
    v-model:value="value"
    :multiple="multiple"
    :loading="roleLoading"
    :options="roleOptions"
    v-bind="$attrs"
    placeholder="请选择角色"
  />
</template>

<style scoped></style>
