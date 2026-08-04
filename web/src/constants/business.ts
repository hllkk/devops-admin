import { transformRecordToOption } from '@/utils/common';

/** 文件上传接受的扩展名（accept） */
export enum AcceptType {
  Image = '.jpg,.jpeg,.png,.gif,.bmp,.webp',
  File = '.doc,.docx,.xls,.xlsx,.ppt,.pptx,.txt,.pdf,.zip,.rar,.7z'
}

/** enable status */
export const enableStatusRecord: Record<Api.Common.EnableStatus, string> = {
  '0': '正常',
  '1': '停用'
};

export const enableStatusOptions = transformRecordToOption(enableStatusRecord);

/** yes or no status */
export const yesOrNoStatusRecord: Record<Api.Common.YesOrNoStatus, string> = {
  Y: '是',
  N: '否'
};

export const yesOrNoStatusOptions = transformRecordToOption(yesOrNoStatusRecord);

/** menu type */
export const menuTypeRecord: Record<Api.System.MenuType, string> = {
  M: '目录',
  C: '菜单',
  F: '按钮'
};

export const menuTypeOptions = transformRecordToOption(menuTypeRecord);

/** menu is frame */
export const menuIsFrameRecord: Record<Api.System.IsMenuFrame, string> = {
  '0': '是',
  '1': '否',
  '2': 'iframe'
};

export const menuIsFrameOptions = transformRecordToOption(menuIsFrameRecord);

/** menu icon type */
export const menuIconTypeRecord: Record<Api.System.IconType, string> = {
  '1': 'iconify',
  '2': '本地图标'
};

export const menuIconTypeOptions = transformRecordToOption(menuIconTypeRecord);

/** menu layout */
export const menuLayoutRecord: Record<Api.System.MenuLayout, string> = {
  '0': '默认布局',
  '1': '空白布局'
};

export const menuLayoutOptions = transformRecordToOption(menuLayoutRecord);

/** data scope（对齐后端 datascope 档位：1全部 2本部门及以下 3本部门 4仅本人 5自定义） */
export const dataScopeRecord: Record<Api.System.DataScope, string> = {
  '1': '全部数据权限',
  '2': '本部门及以下数据权限',
  '3': '本部门数据权限',
  '4': '仅本人数据权限',
  '5': '自定数据权限'
};

export const dataScopeOptions = transformRecordToOption(dataScopeRecord);
