<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

defineOptions({ name: 'SecuritySetting' });

const { t } = useI18n();

const securityConfig = defineModel<Api.System.SecuritySettingConfig>('securityConfig', { required: true });

const captchaTypeOptions = computed(() => [
  { label: t('page.system.setting.captchaType.image'), value: 'image' },
  { label: t('page.system.setting.captchaType.click'), value: 'click' },
  { label: t('page.system.setting.captchaType.slide'), value: 'slide' },
  { label: t('page.system.setting.captchaType.rotate'), value: 'rotate' }
]);

const ipModeOptions = computed(() => [
  { label: t('page.system.setting.ipMode.blacklist'), value: 'blacklist' },
  { label: t('page.system.setting.ipMode.whitelist'), value: 'whitelist' }
]);

const isBlacklistMode = computed(() => securityConfig.value.ipValidationMode === 'blacklist');
const isWhitelistMode = computed(() => securityConfig.value.ipValidationMode === 'whitelist');
</script>

<template>
  <NTabs type="line" animated>
    <!-- 验证码 -->
    <NTabPane name="captcha" :tab="t('page.system.setting.tabCaptcha')">
      <NForm :model="securityConfig" label-placement="left" :label-width="160">
        <NFormItem :label="$t('page.system.setting.captchaEnabled')" path="captchaEnabled">
          <NSwitch v-model:value="securityConfig.captchaEnabled" />
        </NFormItem>
        <template v-if="securityConfig.captchaEnabled">
          <NFormItem :label="$t('page.system.setting.captchaTypeLabel')" path="captchaType">
            <NSelect v-model:value="securityConfig.captchaType" :options="captchaTypeOptions" class="max-w-200px" />
          </NFormItem>
          <NFormItem :label="$t('page.system.setting.captchaOpen')" path="captchaOpen">
            <NInputNumber v-model:value="securityConfig.captchaOpen" :min="0" :max="10" class="max-w-200px">
              <template #suffix>{{ $t('page.system.setting.unitTimes') }}</template>
            </NInputNumber>
          </NFormItem>
          <NFormItem :label="$t('page.system.setting.captchaTimeout')" path="captchaTimeout">
            <NInputNumber v-model:value="securityConfig.captchaTimeout" :min="1" :max="86400" class="max-w-200px">
              <template #suffix>{{ $t('page.system.setting.unitSecond') }}</template>
            </NInputNumber>
          </NFormItem>
          <NFormItem :label="$t('page.system.setting.captchaTolerance')" path="captchaTolerance">
            <NInputNumber v-model:value="securityConfig.captchaTolerance" :min="0" :max="100" class="max-w-200px">
              <template #suffix>{{ $t('page.system.setting.unitPixel') }}</template>
            </NInputNumber>
          </NFormItem>
          <NFormItem :label="$t('page.system.setting.keyLong')" path="keyLong">
            <NInputNumber v-model:value="securityConfig.keyLong" :min="2" :max="8" class="max-w-200px">
              <template #suffix>{{ $t('page.system.setting.unitChar') }}</template>
            </NInputNumber>
          </NFormItem>
          <NFormItem :label="$t('page.system.setting.imgWidth')" path="imgWidth">
            <NInputNumber v-model:value="securityConfig.imgWidth" :min="80" :max="640" class="max-w-200px">
              <template #suffix>{{ $t('page.system.setting.unitPixel') }}</template>
            </NInputNumber>
          </NFormItem>
          <NFormItem :label="$t('page.system.setting.imgHeight')" path="imgHeight">
            <NInputNumber v-model:value="securityConfig.imgHeight" :min="30" :max="400" class="max-w-200px">
              <template #suffix>{{ $t('page.system.setting.unitPixel') }}</template>
            </NInputNumber>
          </NFormItem>
        </template>
      </NForm>
    </NTabPane>

    <!-- 密码复杂度 -->
    <NTabPane name="password" :tab="t('page.system.setting.tabPassword')">
      <NForm :model="securityConfig" label-placement="left" :label-width="160">
        <NFormItem :label="$t('page.system.setting.passwordMinLength')" path="passwordMinLength">
          <NInputNumber v-model:value="securityConfig.passwordMinLength" :min="6" :max="32" class="max-w-200px" />
        </NFormItem>
        <NFormItem :label="$t('page.system.setting.passwordRequireUppercase')" path="passwordRequireUppercase">
          <NSwitch v-model:value="securityConfig.passwordRequireUppercase" />
        </NFormItem>
        <NFormItem :label="$t('page.system.setting.passwordRequireLowercase')" path="passwordRequireLowercase">
          <NSwitch v-model:value="securityConfig.passwordRequireLowercase" />
        </NFormItem>
        <NFormItem :label="$t('page.system.setting.passwordRequireDigit')" path="passwordRequireDigit">
          <NSwitch v-model:value="securityConfig.passwordRequireDigit" />
        </NFormItem>
        <NFormItem :label="$t('page.system.setting.passwordRequireSpecial')" path="passwordRequireSpecial">
          <NSwitch v-model:value="securityConfig.passwordRequireSpecial" />
        </NFormItem>
      </NForm>
    </NTabPane>

    <!-- 限流 -->
    <NTabPane name="limit" :tab="t('page.system.setting.tabLimit')">
      <NForm :model="securityConfig" label-placement="left" :label-width="160">
        <NFormItem :label="$t('page.system.setting.limitEnable')" path="limitEnable">
          <NSwitch v-model:value="securityConfig.limitEnable" />
        </NFormItem>
        <template v-if="securityConfig.limitEnable">
          <NFormItem :label="$t('page.system.setting.limitWindow')" path="limitWindow">
            <NInputNumber v-model:value="securityConfig.limitWindow" :min="1" :max="3600" class="max-w-200px">
              <template #suffix>{{ $t('page.system.setting.unitSecond') }}</template>
            </NInputNumber>
          </NFormItem>
          <NFormItem :label="$t('page.system.setting.limitCount')" path="limitCount">
            <NInputNumber v-model:value="securityConfig.limitCount" :min="1" :max="10000" class="max-w-200px">
              <template #suffix>{{ $t('page.system.setting.unitTimes') }}</template>
            </NInputNumber>
          </NFormItem>
        </template>
      </NForm>
    </NTabPane>

    <!-- 失败锁定 -->
    <NTabPane name="lock" :tab="t('page.system.setting.tabLock')">
      <NForm :model="securityConfig" label-placement="left" :label-width="160">
        <NFormItem :label="$t('page.system.setting.loginFailLockCount')" path="loginFailLockCount">
          <NInputNumber v-model:value="securityConfig.loginFailLockCount" :min="1" :max="10" class="max-w-200px">
            <template #suffix>{{ $t('page.system.setting.unitTimes') }}</template>
          </NInputNumber>
        </NFormItem>
        <NFormItem :label="$t('page.system.setting.loginFailLockTime')" path="loginFailLockTime">
          <NInputNumber v-model:value="securityConfig.loginFailLockTime" :min="1" :max="1440" class="max-w-200px">
            <template #suffix>{{ $t('page.system.setting.unitMinute') }}</template>
          </NInputNumber>
        </NFormItem>
      </NForm>
    </NTabPane>

    <!-- 密码过期 -->
    <NTabPane name="expire" :tab="t('page.system.setting.tabExpire')">
      <NForm :model="securityConfig" label-placement="left" :label-width="160">
        <NFormItem :label="$t('page.system.setting.pwdExpireEnable')" path="pwdExpireEnable">
          <NSwitch v-model:value="securityConfig.pwdExpireEnable" />
        </NFormItem>
        <NFormItem v-if="securityConfig.pwdExpireEnable" :label="$t('page.system.setting.pwdExpireDays')" path="pwdExpireDays">
          <NInputNumber v-model:value="securityConfig.pwdExpireDays" :min="1" :max="365" class="max-w-200px">
            <template #suffix>{{ $t('page.system.setting.unitDay') }}</template>
          </NInputNumber>
        </NFormItem>
      </NForm>
    </NTabPane>

    <!-- 访问控制(IP 校验) -->
    <NTabPane name="access" :tab="t('page.system.setting.tabAccess')">
      <NForm :model="securityConfig" label-placement="left" :label-width="160">
        <NFormItem :label="$t('page.system.setting.ipValidationEnabled')" path="ipValidationEnabled">
          <NSwitch v-model:value="securityConfig.ipValidationEnabled" />
        </NFormItem>
        <template v-if="securityConfig.ipValidationEnabled">
          <NFormItem :label="$t('page.system.setting.ipValidationMode')" path="ipValidationMode">
            <NSelect v-model:value="securityConfig.ipValidationMode" :options="ipModeOptions" class="max-w-200px" />
          </NFormItem>
          <NFormItem :label="$t('page.system.setting.ipBlacklist')" path="ipBlacklist">
            <NInput
              v-model:value="securityConfig.ipBlacklist"
              type="textarea"
              :placeholder="$t('page.system.setting.ipListPlaceholder')"
              :rows="4"
              :disabled="isWhitelistMode"
              class="max-w-400px"
            />
          </NFormItem>
          <NFormItem :label="$t('page.system.setting.ipWhitelist')" path="ipWhitelist">
            <NInput
              v-model:value="securityConfig.ipWhitelist"
              type="textarea"
              :placeholder="$t('page.system.setting.ipListPlaceholder')"
              :rows="4"
              :disabled="isBlacklistMode"
              class="max-w-400px"
            />
          </NFormItem>
        </template>
      </NForm>
    </NTabPane>
  </NTabs>
</template>
