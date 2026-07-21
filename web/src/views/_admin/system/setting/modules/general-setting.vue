<script setup lang="ts">
defineOptions({ name: 'GeneralSetting' });

const config = defineModel<Api.System.GeneralSettingConfig>('config', { required: true });
</script>

<template>
  <NForm :model="config" label-placement="left" :label-width="160">
    <NFormItem :label="$t('page.system.setting.systemName')" path="systemName">
      <NInput v-model:value="config.systemName" :placeholder="$t('page.system.setting.systemName')" class="max-w-400px" />
    </NFormItem>
    <NFormItem :label="$t('page.system.setting.systemDescription')" path="systemDescription">
      <NInput v-model:value="config.systemDescription" :placeholder="$t('page.system.setting.systemDescription')" class="max-w-400px" />
    </NFormItem>
    <NFormItem :label="$t('page.system.setting.logoUrl')" path="logoUrl">
      <div class="flex items-center gap-12px">
        <NInput v-model:value="config.logoUrl" :placeholder="$t('page.system.setting.logoUrl')" class="max-w-360px" />
        <img v-if="config.logoUrl" :src="config.logoUrl" alt="logo" class="max-h-40px max-w-120px rounded-4px" />
      </div>
    </NFormItem>
    <NFormItem :label="$t('page.system.setting.faviconUrl')" path="faviconUrl">
      <div class="flex items-center gap-12px">
        <NInput v-model:value="config.faviconUrl" :placeholder="$t('page.system.setting.faviconUrl')" class="max-w-360px" />
        <img v-if="config.faviconUrl" :src="config.faviconUrl" alt="favicon" class="size-32px rounded-4px" />
      </div>
    </NFormItem>
  </NForm>
</template>
