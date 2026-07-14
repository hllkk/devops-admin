<script setup lang="ts">
import { computed, reactive, ref } from 'vue';
import { useRouter } from 'vue-router';
import { fetchInitDB, fetchPingDB, fetchPingRedis } from '@/service/api';
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
const { createRequiredRule } = useFormRules();

/** stage: notice=须知屏；wizard=三步向导 */
const stage = ref<'notice' | 'wizard'>('notice');
/** 当前向导步骤：1=DB 2=Redis 3=Admin */
const currentStep = ref(1);
const submitting = ref(false);

// 三个独立表单 ref，分步校验（互不干扰）
const { formRef: dbFormRef, validate: validateDB } = useNaiveForm();
const { formRef: redisFormRef, validate: validateRedis } = useNaiveForm();
const { formRef: adminFormRef, validate: validateAdmin } = useNaiveForm();

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
  dbName: 'devops_admin',
  redisAddr: '127.0.0.1:6379',
  redisPassword: '',
  redisDB: 0
});

const isSqlite = computed(() => model.dbType === 'sqlite');
const isPgsql = computed(() => model.dbType === 'pgsql');

// 测试连接 4 态
type TestState = 'idle' | 'testing' | 'success' | 'error';
const dbTest = ref<TestState>('idle');
const redisTest = ref<TestState>('idle');

// 「下一步」是否可点：对应测试通过才放行
const dbStepReady = computed(() => dbTest.value === 'success');
const redisStepReady = computed(() => redisTest.value === 'success');

/** 切换数据库类型时重置连接相关字段默认值 */
function handleDbTypeChange(val: string | number) {
  const type = val as Api.Init.DBType;
  model.dbType = type;
  dbTest.value = 'idle'; // 改动字段 → 失效已测
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

const dbRules = computed(() => {
  const needConn = !isSqlite.value;
  return {
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

const redisRules = {
  redisAddr: [createRequiredRule($t('page.init.rule.redisAddrRequired'))]
};

const adminRules = {
  adminPassword: [createRequiredRule($t('page.init.form.adminPasswordPlaceholder'))]
};

/** 任一相关字段变更 → 失效对应测试标记，强制重测 */
function invalidateOnDbInput() {
  dbTest.value = 'idle';
}
function invalidateOnRedisInput() {
  redisTest.value = 'idle';
}

async function testDB() {
  try {
    await validateDB();
  } catch {
    return;
  }
  dbTest.value = 'testing';
  const { error } = await fetchPingDB({
    dbType: model.dbType,
    host: model.host,
    port: model.port,
    userName: model.userName,
    password: model.password,
    dbName: model.dbName,
    dbPath: model.dbPath,
    template: model.template
  });
  if (error) {
    dbTest.value = 'error';
    return; // 错误文案由请求层统一提示
  }
  dbTest.value = 'success';
  window.$message?.success($t('page.init.testConnectionSuccess'));
}

async function testRedis() {
  try {
    await validateRedis();
  } catch {
    return;
  }
  redisTest.value = 'testing';
  const { error } = await fetchPingRedis({
    addr: model.redisAddr,
    password: model.redisPassword,
    db: model.redisDB
  });
  if (error) {
    redisTest.value = 'error';
    return;
  }
  redisTest.value = 'success';
  window.$message?.success($t('page.init.testConnectionSuccess'));
}

async function nextFromDb() {
  currentStep.value = 2;
}

async function nextFromRedis() {
  currentStep.value = 3;
}

async function handleSubmit() {
  try {
    await validateAdmin();
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
  // 后端初始化后热接入 DB + Redis，免 reload，直接去登录页
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
        <!-- 须知屏 -->
        <div v-if="stage === 'notice'" class="rounded-8px p-16px text-center">
          <h2 class="mb-12px text-22px font-600">{{ $t('page.init.noticeTitle') }}</h2>
          <p class="mb-24px text-15px leading-relaxed color-gray-600 dark:color-gray-300">
            {{ $t('page.init.noticeDesc') }}
          </p>
          <NSpace vertical :size="12">
            <NButton type="primary" size="large" block @click="stage = 'wizard'">
              {{ $t('page.init.confirm') }}
            </NButton>
            <NButton quaternary size="large" block @click="handleBack">
              {{ $t('page.init.back') }}
            </NButton>
          </NSpace>
        </div>

        <!-- 三步向导 -->
        <div v-else>
          <h2 class="mb-16px text-22px font-600">{{ $t('page.init.title') }}</h2>
          <NSteps :current="currentStep" size="small" class="mb-24px">
            <NStep :title="$t('page.init.step.db')" />
            <NStep :title="$t('page.init.step.redis')" />
            <NStep :title="$t('page.init.step.admin')" />
          </NSteps>

          <!-- 步骤 1：数据库 -->
          <NForm
            v-show="currentStep === 1"
            ref="dbFormRef"
            :model="model"
            :rules="dbRules"
            label-placement="top"
            size="large"
          >
            <NFormItem path="dbType" :label="$t('page.init.form.dbType')">
              <NSelect :value="model.dbType" :options="dbTypeOptions" @update:value="handleDbTypeChange" />
            </NFormItem>
            <NFormItem path="dbName" :label="$t('page.init.form.dbName')">
              <NInput
                v-model:value="model.dbName"
                :placeholder="$t('page.init.form.dbNamePlaceholder')"
                @update:value="invalidateOnDbInput"
              />
            </NFormItem>

            <template v-if="!isSqlite">
              <div class="flex gap-16px">
                <NFormItem path="host" :label="$t('page.init.form.host')" class="flex-1">
                  <NInput
                    v-model:value="model.host"
                    :placeholder="$t('page.init.form.hostPlaceholder')"
                    @update:value="invalidateOnDbInput"
                  />
                </NFormItem>
                <NFormItem path="port" :label="$t('page.init.form.port')" class="w-120px">
                  <NInput
                    v-model:value="model.port"
                    :placeholder="$t('page.init.form.portPlaceholder')"
                    @update:value="invalidateOnDbInput"
                  />
                </NFormItem>
              </div>
              <div class="flex gap-16px">
                <NFormItem path="userName" :label="$t('page.init.form.userName')" class="flex-1">
                  <NInput
                    v-model:value="model.userName"
                    :placeholder="$t('page.init.form.userNamePlaceholder')"
                    @update:value="invalidateOnDbInput"
                  />
                </NFormItem>
                <NFormItem path="password" :label="$t('page.init.form.password')" class="flex-1">
                  <NInput
                    v-model:value="model.password"
                    type="password"
                    show-password-on="click"
                    :placeholder="$t('page.init.form.passwordPlaceholder')"
                    @update:value="invalidateOnDbInput"
                  />
                </NFormItem>
              </div>
            </template>

            <NFormItem v-if="isSqlite" path="dbPath" :label="$t('page.init.form.dbPath')">
              <NInput
                v-model:value="model.dbPath"
                :placeholder="$t('page.init.form.dbPathPlaceholder')"
                @update:value="invalidateOnDbInput"
              />
            </NFormItem>

            <NFormItem v-if="isPgsql" path="template" :label="$t('page.init.form.template')">
              <NInput
                v-model:value="model.template"
                :placeholder="$t('page.init.form.templatePlaceholder')"
                @update:value="invalidateOnDbInput"
              />
            </NFormItem>

            <NSpace vertical :size="12" class="mt-8px">
              <NButton
                size="large"
                block
                :loading="dbTest === 'testing'"
                :type="dbTest === 'success' ? 'success' : 'default'"
                @click="testDB"
              >
                {{ $t('page.init.testConnection') }}
              </NButton>
              <NButton type="primary" size="large" block :disabled="!dbStepReady" @click="nextFromDb">
                {{ $t('page.init.next') }}
              </NButton>
              <NButton quaternary size="large" block @click="handleBack">{{ $t('page.init.back') }}</NButton>
            </NSpace>
          </NForm>

          <!-- 步骤 2：Redis -->
          <NForm
            v-show="currentStep === 2"
            ref="redisFormRef"
            :model="model"
            :rules="redisRules"
            label-placement="top"
            size="large"
          >
            <NFormItem path="redisAddr" :label="$t('page.init.form.redisAddr')">
              <NInput
                v-model:value="model.redisAddr"
                :placeholder="$t('page.init.form.redisAddrPlaceholder')"
                @update:value="invalidateOnRedisInput"
              />
            </NFormItem>
            <NFormItem :label="$t('page.init.form.redisPassword')">
              <NInput
                v-model:value="model.redisPassword"
                type="password"
                show-password-on="click"
                :placeholder="$t('page.init.form.redisPasswordPlaceholder')"
                @update:value="invalidateOnRedisInput"
              />
            </NFormItem>
            <NFormItem :label="$t('page.init.form.redisDB')">
              <NInputNumber
                v-model:value="model.redisDB"
                :placeholder="$t('page.init.form.redisDBPlaceholder')"
                class="w-full"
                @update:value="invalidateOnRedisInput"
              />
            </NFormItem>
            <NSpace vertical :size="12" class="mt-8px">
              <NButton
                size="large"
                block
                :loading="redisTest === 'testing'"
                :type="redisTest === 'success' ? 'success' : 'default'"
                @click="testRedis"
              >
                {{ $t('page.init.testConnection') }}
              </NButton>
              <NSpace justify="space-between" :wrap="false">
                <NButton quaternary size="large" @click="currentStep = 1">{{ $t('page.init.prev') }}</NButton>
                <NButton type="primary" size="large" :disabled="!redisStepReady" @click="nextFromRedis">
                  {{ $t('page.init.next') }}
                </NButton>
              </NSpace>
            </NSpace>
          </NForm>

          <!-- 步骤 3：管理员密码 -->
          <NForm
            v-show="currentStep === 3"
            ref="adminFormRef"
            :model="model"
            :rules="adminRules"
            label-placement="top"
            size="large"
          >
            <NFormItem path="adminPassword" :label="$t('page.init.form.adminPassword')">
              <NInput
                v-model:value="model.adminPassword"
                type="password"
                show-password-on="click"
                :placeholder="$t('page.init.form.adminPasswordPlaceholder')"
              />
            </NFormItem>
            <NSpace vertical :size="12" class="mt-8px">
              <NButton type="primary" size="large" block :loading="submitting" @click="handleSubmit">
                {{ $t('page.init.finish') }}
              </NButton>
              <NButton quaternary size="large" block :disabled="submitting" @click="currentStep = 2">
                {{ $t('page.init.prev') }}
              </NButton>
            </NSpace>
          </NForm>
        </div>
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

:deep(.n-input),
:deep(.n-base-selection) {
  --n-height: 42px !important;
  --n-border-radius: 8px !important;
}
</style>
