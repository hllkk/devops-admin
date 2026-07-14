<script setup lang="ts">
import { computed, reactive } from 'vue';
import { useLoading } from '@sa/hooks';
import { fetchRegister } from '@/service/api';
import { useRouterPush } from '@/hooks/common/router';
import { useFormRules, useNaiveForm } from '@/hooks/common/form';
import { $t } from '@/locales';

defineOptions({
  name: 'Register'
});

const { toggleLoginModule } = useRouterPush();
const { formRef, validate } = useNaiveForm();
const { loading: registerLoading, startLoading: startRegisterLoading, endLoading: endRegisterLoading } = useLoading();

const model: Api.Auth.RegisterForm = reactive({
  username: '',
  password: '',
  confirmPassword: '',
  userType: 'sys_user'
});

type RuleKey = Extract<keyof Api.Auth.RegisterForm, 'username' | 'password' | 'confirmPassword'>;

const rules = computed<Record<RuleKey, App.Global.FormRule[]>>(() => {
  const { createConfirmPwdRule, createRequiredRule } = useFormRules();

  return {
    username: [createRequiredRule($t('form.userName.required'))],
    password: [createRequiredRule($t('form.pwd.required'))],
    confirmPassword: createConfirmPwdRule(model.password!)
  };
});

async function handleSubmit() {
  try {
    await validate();
    startRegisterLoading();
    const { error } = await fetchRegister({
      username: model.username,
      password: model.password,
      grantType: 'password',
      userType: model.userType,
      clientId: import.meta.env.VITE_APP_CLIENT_ID
    });
    if (error) {
      return;
    }
    window.$message?.success('注册成功');
    // 注册成功后跳转到登录页
    toggleLoginModule('pwd-login');
  } finally {
    endRegisterLoading();
  }
}
</script>

<template>
  <div>
    <div class="mb-5px text-32px text-black font-600 sm:text-30px dark:text-white">注册新账户</div>
    <div class="pb-18px text-16px text-#858585">欢迎注册！请输入您的账户信息</div>
    <NForm
      ref="formRef"
      :model="model"
      :rules="rules"
      size="large"
      :show-label="false"
      @keyup.enter="() => !registerLoading && handleSubmit()"
    >
      <NFormItem path="username">
        <NInput v-model:value="model.username" :placeholder="$t('page.login.common.userNamePlaceholder')" />
      </NFormItem>
      <NFormItem path="password">
        <NInput
          v-model:value="model.password"
          type="password"
          show-password-on="click"
          :placeholder="$t('page.login.common.passwordPlaceholder')"
        />
      </NFormItem>
      <NFormItem path="confirmPassword">
        <NInput
          v-model:value="model.confirmPassword"
          type="password"
          show-password-on="click"
          :placeholder="$t('page.login.common.confirmPasswordPlaceholder')"
        />
      </NFormItem>
      <NSpace vertical :size="18" class="w-full">
        <NButton type="primary" size="large" block :loading="registerLoading" @click="handleSubmit">
          {{ $t('page.login.common.register') }}
        </NButton>
      </NSpace>
    </NForm>

    <div class="mt-24px w-full text-center text-18px text-#858585">
      您已有账户？
      <NA type="primary" class="text-18px" @click="toggleLoginModule('pwd-login')">
        {{ $t('common.login') }}
      </NA>
    </div>
  </div>
</template>

<style scoped lang="scss">
:deep(.n-base-selection),
:deep(.n-input) {
  --n-height: 42px !important;
  --n-font-size: 16px !important;
  --n-border-radius: 8px !important;
}

:deep(.n-base-selection-label) {
  padding: 0 6px !important;
}

:deep(.n-button) {
  --n-height: 42px !important;
  --n-font-size: 18px !important;
  --n-border-radius: 8px !important;
}
</style>
