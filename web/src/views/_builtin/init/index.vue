<script setup lang="ts">
import { computed, reactive, ref } from 'vue';
import { useRouter } from 'vue-router';
import { fetchInitDB } from '@/service/api';
import { resetSystemInitCheck } from '@/router/guard/route';
import { useFormRules, useNaiveForm } from '@/hooks/common/form';
import { useAppStore } from '@/store/modules/app';
import { useThemeStore } from '@/store/modules/theme';
import loginBackground from '@/assets/svg-icon/login-background.svg';
import { $t } from '@/locales';

defineOptions({
  name: 'Init'
});

const router = useRouter();
const appStore = useAppStore();
const themeStore = useThemeStore();
const { formRef, validate } = useNaiveForm();
const { createRequiredRule } = useFormRules();

/** 是否进入第二步（配置表单） */
const showForm = ref(false);
/** 提交中（控制全屏 loading） */
const submitting = ref(false);

const dbTypeOptions: { label: string; value: Api.Init.DBType }[] = [
  { label: 'MySQL', value: 'mysql' },
  { label: 'PostgreSQL', value: 'pgsql' },
  { label: 'SQLite', value: 'sqlite' },
  { label: 'MSSQL', value: 'mssql' }
];

/** 各数据库类型的连接默认值（sqlite 无需 host/port/user） */
const connDefaults: Record<Exclude<Api.Init.DBType, 'sqlite'>, Partial<Api.Init.InitDBForm>> = {
  mysql: { host: '127.0.0.1', port: '3306', userName: 'root' },
  pgsql: { host: '127.0.0.1', port: '5432', userName: 'postgres', template: 'template0' },
  mssql: { host: '127.0.0.1', port: '1433', userName: 'sa' }
};

const model = reactive<Api.Init.InitDBForm>({
  adminPassword: '',
  dbType: 'mysql',
  host: '127.0.0.1',
  port: '3306',
  userName: 'root',
  password: '',
  dbName: 'devops_admin'
});

const isSqlite = computed(() => model.dbType === 'sqlite');
const isPgsql = computed(() => model.dbType === 'pgsql');

type RuleKey =
  | 'adminPassword'
  | 'dbType'
  | 'dbName'
  | 'host'
  | 'port'
  | 'userName'
  | 'password'
  | 'dbPath'
  | 'template';

const rules = computed<Record<RuleKey, App.Global.FormRule[]>>(() => {
  const needConn = !isSqlite.value;
  return {
    adminPassword: [createRequiredRule($t('page.init.form.adminPasswordPlaceholder'))],
    dbType: [createRequiredRule($t('page.init.form.dbType'))],
    dbName: [createRequiredRule($t('page.init.form.dbNamePlaceholder'))],
    host: needConn ? [createRequiredRule($t('page.init.form.hostPlaceholder'))] : [],
    port: needConn ? [createRequiredRule($t('page.init.form.portPlaceholder'))] : [],
    userName: needConn ? [createRequiredRule($t('page.init.form.userNamePlaceholder'))] : [],
    password: needConn ? [createRequiredRule($t('page.init.form.passwordPlaceholder'))] : [],
    dbPath: isSqlite.value ? [createRequiredRule($t('page.init.form.dbPathPlaceholder'))] : [],
    template: []
  };
});

/** 切换数据库类型时重置连接相关字段默认值 */
function handleDbTypeChange(val: string | number) {
  const type = val as Api.Init.DBType;
  model.dbType = type;
  if (type === 'sqlite') {
    model.host = undefined;
    model.port = undefined;
    model.userName = undefined;
    model.password = undefined;
    model.template = undefined;
    if (model.dbPath === undefined) model.dbPath = '';
    return;
  }
  model.dbPath = undefined;
  Object.assign(model, connDefaults[type]);
  if (type !== 'pgsql') model.template = undefined;
}

async function handleSubmit() {
  try {
    await validate();
  } catch {
    return;
  }
  if (model.adminPassword.trim().length < 6) {
    window.$message?.error($t('page.init.rule.adminPasswordLength'));
    return;
  }
  submitting.value = true;
  const { error } = await fetchInitDB(model);
  submitting.value = false;
  if (error) return; // 错误信息已由请求层统一展示
  resetSystemInitCheck(); // 刷新守卫缓存，避免被重新拦回 /init
  window.$message?.success($t('page.init.successTitle'));
  // 后端初始化后热接入 DB，免 reload，直接去登录页
  router.replace({ name: 'login' });
}

function handleBack() {
  router.replace({ name: 'login' });
}
</script>

<template>
  <div class="scroll box-border size-full flex">
    <div class="relative box-border hidden h-full w-65vw overflow-hidden bg-primary-50 xl:block dark:bg-primary-900">
      <div class="relative z-100 flex items-center pl-30px pt-30px">
        <SystemLogo class="fill-primary size-32px" />
        <h3 class="ml-10px text-20px font-400">{{ $t('system.title') }}</h3>
      </div>
      <div class="absolute inset-x-0 inset-b-10.5% inset-t-0 z-10 m-auto w-40%">
        <img class="size-full" :src="loginBackground" />
      </div>
      <div class="absolute bottom-80px w-full text-center">
        <h1 class="text-24px font-400">{{ $t('page.init.title') }}</h1>
        <p class="mt-8px text-14px color-gray-500">{{ $t('page.init.subTitle') }}</p>
      </div>
      <WaveBg />
    </div>

    <div class="relative h-full flex-1 overflow-auto xl:m-auto sm:!w-full flex flex-col">
      <header class="flex-y-center justify-between px-30px pt-30px xl:justify-end">
        <div class="relative z-100 flex items-center xl:hidden">
          <SystemLogo class="fill-primary size-32px" />
          <h3 class="ml-10px text-20px font-400">{{ $t('system.title') }}</h3>
        </div>
        <div class="flex items-center justify-end">
          <ThemeSchemaSwitch
            :theme-schema="themeStore.themeScheme"
            :show-tooltip="false"
            class="text-20px lt-sm:text-18px"
            @switch="themeStore.toggleThemeScheme"
          />
          <LangSwitch
            v-if="themeStore.header.multilingual.visible"
            :lang="appStore.locale"
            :lang-options="appStore.localeOptions"
            :show-tooltip="false"
            class="text-20px lt-sm:text-18px"
            @change-lang="appStore.changeLocale"
          />
        </div>
      </header>

      <main class="m-auto w-full max-w-560px px-24px">
        <Transition name="fade" mode="out-in">
          <!-- 第一步：初始化须知 -->
          <div v-if="!showForm" key="notice" class="rounded-8px p-16px text-center">
            <h2 class="mb-12px text-22px font-600">{{ $t('page.init.noticeTitle') }}</h2>
            <p class="mb-24px text-15px leading-relaxed color-gray-600 dark:color-gray-300">
              {{ $t('page.init.noticeDesc') }}
            </p>
            <NSpace vertical :size="12">
              <NButton type="primary" size="large" block @click="showForm = true">
                {{ $t('page.init.confirm') }}
              </NButton>
              <NButton quaternary size="large" block @click="handleBack">
                {{ $t('page.init.back') }}
              </NButton>
            </NSpace>
          </div>

          <!-- 第二步：配置表单 -->
          <NForm v-else key="form" ref="formRef" :model="model" :rules="rules" label-placement="top" size="large">
            <h2 class="mb-16px text-22px font-600">{{ $t('page.init.title') }}</h2>

            <NFormItem path="adminPassword" :label="$t('page.init.form.adminPassword')">
              <NInput
                v-model:value="model.adminPassword"
                type="password"
                show-password-on="click"
                :placeholder="$t('page.init.form.adminPasswordPlaceholder')"
              />
            </NFormItem>

            <NFormItem path="dbType" :label="$t('page.init.form.dbType')">
              <NSelect :value="model.dbType" :options="dbTypeOptions" @update:value="handleDbTypeChange" />
            </NFormItem>

            <NFormItem path="dbName" :label="$t('page.init.form.dbName')">
              <NInput v-model:value="model.dbName" :placeholder="$t('page.init.form.dbNamePlaceholder')" />
            </NFormItem>

            <template v-if="!isSqlite">
              <div class="flex gap-16px">
                <NFormItem path="host" :label="$t('page.init.form.host')" class="flex-1">
                  <NInput v-model:value="model.host" :placeholder="$t('page.init.form.hostPlaceholder')" />
                </NFormItem>
                <NFormItem path="port" :label="$t('page.init.form.port')" class="w-120px">
                  <NInput v-model:value="model.port" :placeholder="$t('page.init.form.portPlaceholder')" />
                </NFormItem>
              </div>
              <div class="flex gap-16px">
                <NFormItem path="userName" :label="$t('page.init.form.userName')" class="flex-1">
                  <NInput v-model:value="model.userName" :placeholder="$t('page.init.form.userNamePlaceholder')" />
                </NFormItem>
                <NFormItem path="password" :label="$t('page.init.form.password')" class="flex-1">
                  <NInput
                    v-model:value="model.password"
                    type="password"
                    show-password-on="click"
                    :placeholder="$t('page.init.form.passwordPlaceholder')"
                  />
                </NFormItem>
              </div>
            </template>

            <NFormItem v-if="isSqlite" path="dbPath" :label="$t('page.init.form.dbPath')">
              <NInput v-model:value="model.dbPath" :placeholder="$t('page.init.form.dbPathPlaceholder')" />
            </NFormItem>

            <NFormItem v-if="isPgsql" path="template" :label="$t('page.init.form.template')">
              <NInput v-model:value="model.template" :placeholder="$t('page.init.form.templatePlaceholder')" />
            </NFormItem>

            <NSpace vertical :size="12" class="mt-8px">
              <NButton type="primary" size="large" block :loading="submitting" @click="handleSubmit">
                {{ $t('page.init.submit') }}
              </NButton>
              <NButton quaternary size="large" block :disabled="submitting" @click="handleBack">
                {{ $t('page.init.back') }}
              </NButton>
            </NSpace>
          </NForm>
        </Transition>
      </main>
    </div>

    <!-- 全屏 loading（单文案） -->
    <Teleport to="body">
      <div
        v-if="submitting"
        class="fixed inset-0 z-9999 flex flex-col items-center justify-center gap-16px bg-black/70"
      >
        <NSpin :size="48" />
        <span class="text-16px text-white">{{ $t('page.init.submitting') }}</span>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.scroll {
  overflow: auto;
}

.scroll::-webkit-scrollbar {
  display: none;
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

:deep(.n-input),
:deep(.n-base-selection) {
  --n-height: 42px !important;
  --n-border-radius: 8px !important;
}
</style>
