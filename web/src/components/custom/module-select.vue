<script setup lang="ts">
import { computed, ref } from 'vue';
import type { SelectOption } from 'naive-ui';
import { useLoading } from '@sa/hooks';
import { useAuthStore } from '@/store/modules/auth';


defineOptions({ name: 'ModuleSelect' });

interface Props {
  clearable?: boolean;
}

withDefaults(defineProps<Props>(), {
  clearable: false
});
const { userInfo } = useAuthStore();

const { loading } = useLoading();

const moduleId = defineModel<CommonType.IdType>('moduleId', { required: false, default: undefined });
const enabled = defineModel<boolean>('enabled', { required: false, default: false });
const moduleOption = ref<SelectOption[]>([]);

const handleChangeModule = (val: CommonType.IdType) => {
  moduleId.value = val;
};

const handleClearModule = () => {

};



const showModuleSelect = computed<boolean>(() => {
  return userInfo.user?.userId === 1 && enabled.value;
});
</script>

<template>
  <NSelect
    v-if="showModuleSelect"
    v-model:value="moduleId"
    :clearable="clearable"
    placeholder="请选择模块"
    :options="moduleOption"
    :loading="loading"
    @update:value="handleChangeModule"
    @clear="handleClearModule"
  />
</template>
