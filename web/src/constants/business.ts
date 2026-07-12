import { transformRecordToOption } from '@/utils/common';

/** enable status */
export const enableStatusRecord: Record<Api.Common.EnableStatus, string> = {
  '0': '正常',
  '1': '停用'
};

export const enableStatusOptions = transformRecordToOption(enableStatusRecord);
