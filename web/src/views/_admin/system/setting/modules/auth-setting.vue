<script setup lang="ts">
import { useI18n } from 'vue-i18n';

defineOptions({ name: 'AuthSetting' });

const { t } = useI18n();

const configModel = defineModel<Api.System.AuthSettingConfig>('config', { required: true });
</script>

<template>
  <NTabs type="line" animated>
    <!-- 账号功能 -->
    <NTabPane name="account" :tab="t('page.system.setting.tabAccountFunction')">
      <NForm :model="configModel" label-placement="left" :label-width="180" class="mt-16px">
        <NFormItem :label="$t('page.system.setting.authRegisterEnabled')" path="registerEnabled">
          <NSwitch v-model:value="configModel.registerEnabled" />
          <span class="ml-12px text-12px color-gray-400">{{ $t('page.system.setting.authRegisterEnabledTip') }}</span>
        </NFormItem>
        <NFormItem :label="$t('page.system.setting.authResetPwdEnabled')" path="resetPwdEnabled">
          <NSwitch v-model:value="configModel.resetPwdEnabled" />
          <span class="ml-12px text-12px color-gray-400">{{ $t('page.system.setting.authResetPwdEnabledTip') }}</span>
        </NFormItem>
      </NForm>
    </NTabPane>

    <!-- 企业微信 -->
    <NTabPane name="wecom" :tab="$t('page.system.setting.tabAuthWecom')">
      <NForm :model="configModel" label-placement="left" :label-width="180" class="mt-16px">
        <NFormItem :label="$t('page.system.setting.authOAuthEnabled')" path="wecomEnabled">
          <NSwitch v-model:value="configModel.wecomEnabled" />
        </NFormItem>
        <template v-if="configModel.wecomEnabled">
          <NFormItem :label="$t('page.system.setting.authWecomCorpId')" path="wecomCorpId">
            <NInput
              v-model:value="configModel.wecomCorpId"
              :placeholder="$t('page.system.setting.authWecomCorpIdPlaceholder')"
              class="max-w-400px"
            />
          </NFormItem>
          <NFormItem :label="$t('page.system.setting.authWecomAgentId')" path="wecomAgentId">
            <NInputNumber
              v-model:value="configModel.wecomAgentId"
              :placeholder="$t('page.system.setting.authWecomAgentIdPlaceholder')"
              class="max-w-400px"
              :show-button="false"
            />
          </NFormItem>
          <NFormItem :label="$t('page.system.setting.authOAuthClientSecret')" path="wecomClientSecret">
            <NInput
              v-model:value="configModel.wecomClientSecret"
              type="password"
              show-password-on="click"
              :placeholder="$t('page.system.setting.authOAuthClientSecretPlaceholder')"
              class="max-w-400px"
            />
          </NFormItem>
          <NFormItem :label="$t('page.system.setting.authOAuthCallbackUrl')" path="wecomCallbackUrl">
            <div class="flex flex-col gap-4px max-w-400px">
              <NInput
                v-model:value="configModel.wecomCallbackUrl"
                :placeholder="$t('page.system.setting.authOAuthCallbackUrlPlaceholder')"
              />
              <span class="text-12px color-gray-400">{{ $t('page.system.setting.authOAuthCallbackUrlTip') }}</span>
            </div>
          </NFormItem>

          <NCollapse class="mt-16px">
            <NCollapseItem :title="$t('page.system.setting.authWecomDomainVerifyTitle')" name="domain">
              <NAlert type="warning" :show-icon="false" class="mb-16px">
                <div>{{ $t('page.system.setting.authWecomDomainVerifyTip1') }}</div>
                <div>{{ $t('page.system.setting.authWecomDomainVerifyTip2') }}</div>
                <div>{{ $t('page.system.setting.authWecomDomainVerifyTip3') }}</div>
              </NAlert>
              <NFormItem :label="$t('page.system.setting.authWecomDomainFileName')" path="wecomDomainFileName">
                <NInput
                  v-model:value="configModel.wecomDomainFileName"
                  :placeholder="$t('page.system.setting.authWecomDomainFileNamePlaceholder')"
                  class="max-w-400px"
                />
              </NFormItem>
              <NFormItem :label="$t('page.system.setting.authWecomDomainFileContent')" path="wecomDomainFileContent">
                <NInput
                  v-model:value="configModel.wecomDomainFileContent"
                  :placeholder="$t('page.system.setting.authWecomDomainFileContentPlaceholder')"
                  class="max-w-400px"
                />
              </NFormItem>
            </NCollapseItem>
          </NCollapse>
        </template>
      </NForm>
    </NTabPane>

    <!-- 微信开放平台 -->
    <NTabPane name="wechat" :tab="$t('page.system.setting.tabAuthWechat')">
      <NForm :model="configModel" label-placement="left" :label-width="180" class="mt-16px">
        <NFormItem :label="$t('page.system.setting.authOAuthEnabled')" path="wechatEnabled">
          <NSwitch v-model:value="configModel.wechatEnabled" />
        </NFormItem>
        <template v-if="configModel.wechatEnabled">
          <NFormItem :label="$t('page.system.setting.authOAuthClientId')" path="wechatClientId">
            <NInput
              v-model:value="configModel.wechatClientId"
              :placeholder="$t('page.system.setting.authOAuthClientIdPlaceholder')"
              class="max-w-400px"
            />
          </NFormItem>
          <NFormItem :label="$t('page.system.setting.authOAuthClientSecret')" path="wechatClientSecret">
            <NInput
              v-model:value="configModel.wechatClientSecret"
              type="password"
              show-password-on="click"
              :placeholder="$t('page.system.setting.authOAuthClientSecretPlaceholder')"
              class="max-w-400px"
            />
          </NFormItem>
          <NFormItem :label="$t('page.system.setting.authOAuthCallbackUrl')" path="wechatCallbackUrl">
            <div class="flex flex-col gap-4px max-w-400px">
              <NInput
                v-model:value="configModel.wechatCallbackUrl"
                :placeholder="$t('page.system.setting.authOAuthCallbackUrlPlaceholder')"
              />
              <span class="text-12px color-gray-400">{{ $t('page.system.setting.authOAuthCallbackUrlTip') }}</span>
            </div>
          </NFormItem>
        </template>
      </NForm>
    </NTabPane>

    <!-- Gitee -->
    <NTabPane name="gitee" :tab="$t('page.system.setting.tabAuthGitee')">
      <NForm :model="configModel" label-placement="left" :label-width="180" class="mt-16px">
        <NFormItem :label="$t('page.system.setting.authOAuthEnabled')" path="giteeEnabled">
          <NSwitch v-model:value="configModel.giteeEnabled" />
        </NFormItem>
        <template v-if="configModel.giteeEnabled">
          <NFormItem :label="$t('page.system.setting.authOAuthClientId')" path="giteeClientId">
            <NInput
              v-model:value="configModel.giteeClientId"
              :placeholder="$t('page.system.setting.authOAuthClientIdPlaceholder')"
              class="max-w-400px"
            />
          </NFormItem>
          <NFormItem :label="$t('page.system.setting.authOAuthClientSecret')" path="giteeClientSecret">
            <NInput
              v-model:value="configModel.giteeClientSecret"
              type="password"
              show-password-on="click"
              :placeholder="$t('page.system.setting.authOAuthClientSecretPlaceholder')"
              class="max-w-400px"
            />
          </NFormItem>
          <NFormItem :label="$t('page.system.setting.authOAuthCallbackUrl')" path="giteeCallbackUrl">
            <div class="flex flex-col gap-4px max-w-400px">
              <NInput
                v-model:value="configModel.giteeCallbackUrl"
                :placeholder="$t('page.system.setting.authOAuthCallbackUrlPlaceholder')"
              />
              <span class="text-12px color-gray-400">{{ $t('page.system.setting.authOAuthCallbackUrlTip') }}</span>
            </div>
          </NFormItem>
        </template>
      </NForm>
    </NTabPane>

    <!-- GitHub -->
    <NTabPane name="github" :tab="$t('page.system.setting.tabAuthGithub')">
      <NForm :model="configModel" label-placement="left" :label-width="180" class="mt-16px">
        <NFormItem :label="$t('page.system.setting.authOAuthEnabled')" path="githubEnabled">
          <NSwitch v-model:value="configModel.githubEnabled" />
        </NFormItem>
        <template v-if="configModel.githubEnabled">
          <NFormItem :label="$t('page.system.setting.authOAuthClientId')" path="githubClientId">
            <NInput
              v-model:value="configModel.githubClientId"
              :placeholder="$t('page.system.setting.authOAuthClientIdPlaceholder')"
              class="max-w-400px"
            />
          </NFormItem>
          <NFormItem :label="$t('page.system.setting.authOAuthClientSecret')" path="githubClientSecret">
            <NInput
              v-model:value="configModel.githubClientSecret"
              type="password"
              show-password-on="click"
              :placeholder="$t('page.system.setting.authOAuthClientSecretPlaceholder')"
              class="max-w-400px"
            />
          </NFormItem>
          <NFormItem :label="$t('page.system.setting.authOAuthCallbackUrl')" path="githubCallbackUrl">
            <div class="flex flex-col gap-4px max-w-400px">
              <NInput
                v-model:value="configModel.githubCallbackUrl"
                :placeholder="$t('page.system.setting.authOAuthCallbackUrlPlaceholder')"
              />
              <span class="text-12px color-gray-400">{{ $t('page.system.setting.authOAuthCallbackUrlTip') }}</span>
            </div>
          </NFormItem>
        </template>
      </NForm>
    </NTabPane>

    <!-- 钉钉 -->
    <NTabPane name="dingtalk" :tab="$t('page.system.setting.tabAuthDingtalk')">
      <NForm :model="configModel" label-placement="left" :label-width="180" class="mt-16px">
        <NFormItem :label="$t('page.system.setting.authOAuthEnabled')" path="dingtalkEnabled">
          <NSwitch v-model:value="configModel.dingtalkEnabled" />
        </NFormItem>
        <template v-if="configModel.dingtalkEnabled">
          <NFormItem :label="$t('page.system.setting.authOAuthClientId')" path="dingtalkClientId">
            <NInput
              v-model:value="configModel.dingtalkClientId"
              :placeholder="$t('page.system.setting.authOAuthClientIdPlaceholder')"
              class="max-w-400px"
            />
          </NFormItem>
          <NFormItem :label="$t('page.system.setting.authOAuthClientSecret')" path="dingtalkClientSecret">
            <NInput
              v-model:value="configModel.dingtalkClientSecret"
              type="password"
              show-password-on="click"
              :placeholder="$t('page.system.setting.authOAuthClientSecretPlaceholder')"
              class="max-w-400px"
            />
          </NFormItem>
          <NFormItem :label="$t('page.system.setting.authOAuthCallbackUrl')" path="dingtalkCallbackUrl">
            <div class="flex flex-col gap-4px max-w-400px">
              <NInput
                v-model:value="configModel.dingtalkCallbackUrl"
                :placeholder="$t('page.system.setting.authOAuthCallbackUrlPlaceholder')"
              />
              <span class="text-12px color-gray-400">{{ $t('page.system.setting.authOAuthCallbackUrlTip') }}</span>
            </div>
          </NFormItem>
        </template>
      </NForm>
    </NTabPane>
  </NTabs>
</template>
