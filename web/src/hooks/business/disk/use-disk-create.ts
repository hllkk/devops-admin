import { useMessage } from 'naive-ui';
import { useDiskStore } from '@/store/modules/disk';
import { fetchCreateFile, fetchMkdir, fetchRename } from '@/service/api/disk/file';
import { resolveNameConflict } from '@/utils/disk';
import { $t } from '@/locales';

// 列表刷新回调(由 index.vue 注入 getFileList)。模块级共享,使 begin/submit 在不同组件实例间一致。
let refreshFn: (() => void) | null = null;
// 提交锁:防止 blur 与 Enter / 确认按钮并发重复提交
let submitting = false;

/**
 * 行内新建 / 重命名。
 *
 * 状态收口在 diskStore(creatingType / renamingId / creatingName),列表与网格视图共用同一份;
 * 实际建文件夹 / 建空文件 / 改名请求 + 列表刷新由此 composable 承担。
 * - 确认:Enter / ✓ 按钮 / 失焦(延迟 150ms)
 * - 取消:Esc / ✗ 按钮
 * - 同名:静默自动加 (n) 后缀(复用 resolveNameConflict)
 */
export function useDiskCreate() {
  const diskStore = useDiskStore();
  const message = useMessage();

  function registerRefresh(fn: () => void) {
    refreshFn = fn;
  }

  function defaultName(type: 'file' | 'folder') {
    return $t(type === 'file' ? 'page.disk.createInline.defaultFileName' : 'page.disk.createInline.defaultFolderName');
  }

  function beginCreate(type: 'file' | 'folder') {
    diskStore.startCreating(type);
    diskStore.setCreatingName(defaultName(type));
  }

  function beginRename(file: Api.Disk.FileItem) {
    diskStore.startRenaming(file);
  }

  function cancel() {
    diskStore.cancelCreating();
  }

  async function submit() {
    if (submitting) return;
    const isRename = !!diskStore.renamingId;
    const type = diskStore.creatingType;
    if (!type && !isRename) return;

    const name = (diskStore.creatingName || '').trim();
    if (!name) {
      message.warning($t('page.disk.msg.nameRequired'));
      return;
    }

    // 同级同名:静默加 (n) 后缀
    const existing = diskStore.currentFileList.map(f => f.fileName);
    const finalName = existing.includes(name) ? resolveNameConflict(name, existing) : name;

    submitting = true;
    try {
      const parentPath = diskStore.getCurrentPathString();
      let error: unknown;
      if (isRename && diskStore.renamingId) {
        ({ error } = await fetchRename({ fileId: diskStore.renamingId, newName: finalName }));
      } else if (type === 'file') {
        ({ error } = await fetchCreateFile({ parentPath, fileName: finalName }));
      } else {
        ({ error } = await fetchMkdir({ parentPath, folderName: finalName }));
      }
      if (error) {
        message.error($t('page.disk.msg.operateFail'));
        return;
      }
      message.success($t(isRename ? 'page.disk.msg.renameSuccess' : 'page.disk.msg.newFolderSuccess'));
      diskStore.cancelCreating();
      refreshFn?.();
    } finally {
      submitting = false;
    }
  }

  function keydown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      e.preventDefault();
      submit();
    } else if (e.key === 'Escape') {
      cancel();
    }
  }

  // 失焦自动确认(延迟,让确认/取消按钮的 mousedown 先触发,避免按钮失效)
  function blur() {
    setTimeout(() => {
      if (submitting) return;
      if (!diskStore.creatingType && !diskStore.renamingId) return;
      submit();
    }, 150);
  }

  return { beginCreate, beginRename, cancel, submit, keydown, blur, registerRefresh };
}
