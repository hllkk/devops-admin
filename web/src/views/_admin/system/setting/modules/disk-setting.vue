<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

defineOptions({ name: 'DiskSetting' });

const { t } = useI18n();

const configModel = defineModel<Api.System.DiskSettingConfig>('config', { required: true });

const sizeUnitOptions = computed(() => [
  { label: t('page.system.setting.diskUnitMB'), value: 'MB' },
  { label: t('page.system.setting.diskUnitGB'), value: 'GB' },
  { label: t('page.system.setting.diskUnitTB'), value: 'TB' }
]);
</script>

<template>
  <NTabs type="line" animated>
    <!-- 基础配置 -->
    <NTabPane name="basic" :tab="t('page.system.setting.tabDiskBasic')">
      <NForm :model="configModel" label-placement="left" :label-width="160">
        <NFormItem :label="$t('page.system.setting.diskMaxUploadSize')" path="maxUploadSize">
          <NFlex align="center" :wrap="false" class="gap-8px max-w-360px">
            <NInputNumber v-model:value="configModel.maxUploadSize" :min="1" :precision="0" class="flex-1" />
            <NSelect v-model:value="configModel.maxUploadSizeUnit" :options="sizeUnitOptions" class="w-80px" />
          </NFlex>
        </NFormItem>
        <NFormItem :label="$t('page.system.setting.diskStorageQuota')" path="storageQuota">
          <NFlex align="center" :wrap="false" class="gap-8px max-w-360px">
            <NInputNumber v-model:value="configModel.storageQuota" :min="1" :precision="0" class="flex-1" />
            <NSelect v-model:value="configModel.storageQuotaUnit" :options="sizeUnitOptions" class="w-80px" />
          </NFlex>
        </NFormItem>
        <NFormItem :label="$t('page.system.setting.diskMaxConcurrentUploads')" path="maxConcurrentUploads">
          <NInputNumber v-model:value="configModel.maxConcurrentUploads" :min="0" :max="50" :precision="0" class="max-w-200px">
            <template #suffix>{{ $t('page.system.setting.diskMaxConcurrentUploadsSuffix') }}</template>
          </NInputNumber>
        </NFormItem>
        <NFormItem :label="$t('page.system.setting.diskMaxChunkConcurrency')" path="maxChunkConcurrency">
          <NInputNumber v-model:value="configModel.maxChunkConcurrency" :min="0" :max="16" :precision="0" class="max-w-200px">
            <template #suffix>{{ $t('page.system.setting.diskMaxChunkConcurrencySuffix') }}</template>
          </NInputNumber>
        </NFormItem>
        <NFormItem :label="$t('page.system.setting.diskMaxChunkRetries')" path="maxChunkRetries">
          <NInputNumber v-model:value="configModel.maxChunkRetries" :min="0" :max="10" :precision="0" class="max-w-200px">
            <template #suffix>{{ $t('page.system.setting.diskMaxChunkRetriesSuffix') }}</template>
          </NInputNumber>
        </NFormItem>
        <NFormItem path="allowedExtensions">
          <template #label>
            <NTooltip trigger="hover">
              <template #trigger>
                <span class="flex items-center gap-4px cursor-help">
                  {{ $t('page.system.setting.diskAllowedExtensions') }}
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    width="14"
                    height="14"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    class="text-gray-400"
                  >
                    <circle cx="12" cy="12" r="10" />
                    <path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3" />
                    <path d="M12 17h.01" />
                  </svg>
                </span>
              </template>
              {{ $t('page.system.setting.diskAllowedExtensionsTip') }}
            </NTooltip>
          </template>
          <NInput
            v-model:value="configModel.allowedExtensions"
            :placeholder="t('page.system.setting.diskAllowedExtensionsTip')"
            class="max-w-400px"
          />
        </NFormItem>
        <NFormItem path="blockedExtensions">
          <template #label>
            <NTooltip trigger="hover">
              <template #trigger>
                <span class="flex items-center gap-4px cursor-help">
                  {{ $t('page.system.setting.diskBlockedExtensions') }}
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    width="14"
                    height="14"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    class="text-gray-400"
                  >
                    <circle cx="12" cy="12" r="10" />
                    <path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3" />
                    <path d="M12 17h.01" />
                  </svg>
                </span>
              </template>
              {{ $t('page.system.setting.diskBlockedExtensionsTip') }}
            </NTooltip>
          </template>
          <NInput
            v-model:value="configModel.blockedExtensions"
            :placeholder="t('page.system.setting.diskBlockedExtensionsTip')"
            class="max-w-400px"
          />
        </NFormItem>
        <NFormItem :label="$t('page.system.setting.diskRecycleBinRetentionDays')" path="recycleBinRetentionDays">
          <NInputNumber v-model:value="configModel.recycleBinRetentionDays" :min="1" :max="365" class="max-w-200px">
            <template #suffix>{{ $t('page.system.setting.unitDay') }}</template>
          </NInputNumber>
        </NFormItem>
      </NForm>
    </NTabPane>

    <!-- 个性化 -->
    <NTabPane name="display" :tab="t('page.system.setting.tabDiskDisplay')">
      <NForm :model="configModel" label-placement="left" :label-width="160">
        <NFormItem :label="$t('page.system.setting.diskName')" path="diskName">
          <NInput
            v-model:value="configModel.diskName"
            :placeholder="$t('page.system.setting.diskNamePlaceholder')"
            class="max-w-400px"
          />
        </NFormItem>
        <NFormItem :label="$t('page.system.setting.diskLogo')" path="diskLogo">
          <NInput
            v-model:value="configModel.diskLogo"
            :placeholder="$t('page.system.setting.diskLogoPlaceholder')"
            class="max-w-400px"
          />
        </NFormItem>
      </NForm>
    </NTabPane>

    <!-- OnlyOffice -->
    <NTabPane name="onlyOffice" :tab="t('page.system.setting.tabDiskOnlyOffice')">
      <NForm :model="configModel" label-placement="left" :label-width="160">
        <NFormItem :label="$t('page.system.setting.diskOnlyOfficeEnabled')" path="onlyOfficeEnabled">
          <NSwitch v-model:value="configModel.onlyOfficeEnabled" />
        </NFormItem>
        <template v-if="configModel.onlyOfficeEnabled">
          <NFormItem path="onlyOfficeServerUrl">
            <template #label>
              <NTooltip trigger="hover">
                <template #trigger>
                  <span class="flex items-center gap-4px cursor-help">
                    {{ $t('page.system.setting.diskOnlyOfficeServerUrl') }}
                    <svg
                      xmlns="http://www.w3.org/2000/svg"
                      width="14"
                      height="14"
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="currentColor"
                      stroke-width="2"
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      class="text-gray-400"
                    >
                      <circle cx="12" cy="12" r="10" />
                      <path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3" />
                      <path d="M12 17h.01" />
                    </svg>
                  </span>
                </template>
                {{ $t('page.system.setting.diskOnlyOfficeServerUrlTip') }}
              </NTooltip>
            </template>
            <NInput
              v-model:value="configModel.onlyOfficeServerUrl"
              :placeholder="$t('page.system.setting.diskOnlyOfficeServerUrlPlaceholder')"
              class="max-w-400px"
            />
          </NFormItem>
          <NFormItem path="onlyOfficeTokenSecret">
            <template #label>
              <NTooltip trigger="hover">
                <template #trigger>
                  <span class="flex items-center gap-4px cursor-help">
                    {{ $t('page.system.setting.diskOnlyOfficeTokenSecret') }}
                    <svg
                      xmlns="http://www.w3.org/2000/svg"
                      width="14"
                      height="14"
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="currentColor"
                      stroke-width="2"
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      class="text-gray-400"
                    >
                      <circle cx="12" cy="12" r="10" />
                      <path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3" />
                      <path d="M12 17h.01" />
                    </svg>
                  </span>
                </template>
                {{ $t('page.system.setting.diskOnlyOfficeTokenSecretTip') }}
              </NTooltip>
            </template>
            <NInput
              v-model:value="configModel.onlyOfficeTokenSecret"
              type="password"
              show-password-on="click"
              :placeholder="$t('page.system.setting.diskOnlyOfficeTokenSecretPlaceholder')"
              class="max-w-400px"
            />
          </NFormItem>
          <NFormItem path="onlyOfficeCallbackUrl">
            <template #label>
              <NTooltip trigger="hover">
                <template #trigger>
                  <span class="flex items-center gap-4px cursor-help">
                    {{ $t('page.system.setting.diskOnlyOfficeCallbackUrl') }}
                    <svg
                      xmlns="http://www.w3.org/2000/svg"
                      width="14"
                      height="14"
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="currentColor"
                      stroke-width="2"
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      class="text-gray-400"
                    >
                      <circle cx="12" cy="12" r="10" />
                      <path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3" />
                      <path d="M12 17h.01" />
                    </svg>
                  </span>
                </template>
                {{ $t('page.system.setting.diskOnlyOfficeCallbackUrlTip') }}
              </NTooltip>
            </template>
            <NInput
              v-model:value="configModel.onlyOfficeCallbackUrl"
              :placeholder="$t('page.system.setting.diskOnlyOfficeCallbackUrlPlaceholder')"
              class="max-w-400px"
            />
          </NFormItem>
        </template>
      </NForm>
    </NTabPane>
  </NTabs>
</template>
