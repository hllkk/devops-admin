/**
 * Namespace Api
 *
 * All backend api type
 */
declare namespace Api {
  /**
   * namespace Log
   *
   * backend api module: "log"
   */
  namespace Log {
    /** 业务操作类型 */
    type BusinessType = '0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9';

    /** oper log */
    type OperLog = Common.CommonRecord<{
      /** 日志主键 */
      operId: CommonType.IdType;
      /** 系统模块 */
      title: string;
      /** 操作类型 */
      businessType: Log.BusinessType;
      /** 方法名称 */
      method: string;
      /** 请求方式 */
      requestMethod: string;
      /** 操作类别 */
      operatorType: string;
      /** 操作人员 */
      operName: string;
      /** 部门名称 */
      deptName: string;
      /** 请求URL */
      operUrl: string;
      /** 操作IP */
      operIp: string;
      /** 操作地点 */
      operLocation: string;
      /** 请求参数 */
      operParam: string;
      /** 返回参数 */
      jsonResult: string;
      /** 操作状态 */
      status: Common.EnableStatus;
      /** 错误消息 */
      errorMsg: string;
      /** 操作时间 */
      operTime: string;
      /** 消耗时间 */
      costTime: number;
    }>;

    /** oper log search params */
    type OperLogSearchParams = CommonType.RecordNullable<
      Pick<Api.Log.OperLog, 'title' | 'businessType' | 'operName' | 'operIp' | 'status' | 'operTime'> &
        Api.Common.CommonSearchParams
    >;

    /** oper log list */
    type OperLogList = Api.Common.PaginatingQueryRecord<OperLog>;

    /** login log */
    type LoginLog = Common.CommonRecord<{
      /** 访问ID */
      infoId: CommonType.IdType;
      /** 用户账号 */
      userName: string;
      /** 客户端 */
      clientKey: string;
      /** 设备类型 */
      deviceType: System.DeviceType;
      /** 登录IP地址 */
      ipaddr: string;
      /** 登录地点 */
      loginLocation: string;
      /** 浏览器类型 */
      browser: string;
      /** 操作系统 */
      os: string;
      /** 登录状态（0成功 1失败） */
      status: Common.EnableStatus;
      /** 提示消息 */
      msg: string;
      /** 访问时间 */
      loginTime: string;
    }>;

    /** login log search params */
    type LoginLogSearchParams = CommonType.RecordNullable<
      Pick<Api.Log.LoginLog, 'userName' | 'ipaddr' | 'status'> & Api.Common.CommonSearchParams
    >;

    /** login log list */
    type LoginLogList = Api.Common.PaginatingQueryRecord<LoginLog>;

    /** error log */
    type ErrorLog = {
      /** 主键 */
      id: string;
      /** 创建时间 */
      createTime: string;
      /** 更新时间 */
      updateTime: string;
      /** 错误来源 */
      form: string;
      /** 错误内容 */
      info: string;
      /** 日志等级 */
      level: string;
      /** 请求ID */
      request_id: string;
      /** 链路ID */
      trace_id: string;
      /** 解决方案 */
      solution: string;
      /** 处理状态 */
      status: string;
    };

    /** error log search params */
    type ErrorLogSearchParams = CommonType.RecordNullable<
      Pick<Api.Log.ErrorLog, 'form' | 'info' | 'level' | 'status'> & {
        pageNum: number;
        pageSize: number;
        createdAtRange: [string, string] | null;
      }
    >;

    /** error log list */
    type ErrorLogList = Api.Common.PaginatingQueryRecord<ErrorLog>;
  }
}
