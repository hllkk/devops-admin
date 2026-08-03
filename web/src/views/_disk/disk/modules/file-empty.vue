<script setup lang="ts">
import { $t } from '@/locales';

defineOptions({ name: 'FileEmpty' });
</script>

<template>
  <!-- 空文件夹：grid 与 list 共用同一空状态，保证两视图展示一致 -->
  <div class="flex-col-center gap-12px">
    <SvgIcon local-icon="empty-data" class="text-98" />
    <span class="text-14px">{{ $t('page.disk.empty') }}</span>
  </div>
</template>
