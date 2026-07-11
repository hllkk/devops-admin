/**
 * Namespace Api
 *
 * All backend api type
 */

declare namespace Api {
  /**
   * namespace Monitor
   *
   * backend api module: "monitor"
   */
  namespace Monitor {
    type OnlineUser = Common.CommonRecord<{
      /** 用户账号 */
      userName: string;
      /** 登录IP地址 */
      ipaddr: string;
      /** 登录地点 */
      loginLocation: string;
      /** 浏览器类型 */
      browser: string;
      /** 操作系统 */
      os: string;
      /** 所在部门 */
      deptName: string;
      /** 设备类型 */
      deviceType: System.DeviceType;
      /** 登录时间 */
      loginTime: number;
      /** 令牌ID */
      tokenId: string;
    }>;

    /** online user list */
    type OnlineUserList = Api.Common.PaginatingQueryRecord<OnlineUser>;

     /** online user search params */
    type OnlineUserSearchParams = CommonType.RecordNullable<
      Pick<Api.Monitor.OnlineUser, 'userName' | 'ipaddr'> & Api.Common.CommonSearchParams
    >;
  }
}
