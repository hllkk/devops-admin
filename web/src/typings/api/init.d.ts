declare namespace Api {
  /**
   * namespace Init
   *
   * backend api module: "init"（系统初始化 / /init/* ）
   */
  namespace Init {
    /** 数据库类型 */
    type DBType = 'mysql' | 'pgsql' | 'sqlite' | 'mssql';

    /** /init/checkdb 响应：是否需要初始化 */
    interface CheckDBResult {
      /** true 表示尚未初始化，需要前往初始化 */
      needInit: boolean;
    }

    /**
     * /init/initdb 请求体
     *
     * 字段与后端 request.InitDB 对齐：
     * - sqlite 仅需 dbPath + dbName；host/port/userName/password 不传
     * - pgsql 可选 template
     */
    interface InitDBForm {
      /** 管理员密码（≥6 位） */
      adminPassword: string;
      /** 数据库类型 */
      dbType: DBType;
      /** 数据库地址 */
      host?: string;
      /** 数据库端口 */
      port?: string;
      /** 数据库用户名 */
      userName?: string;
      /** 数据库密码 */
      password?: string;
      /** 数据库名（必填） */
      dbName: string;
      /** sqlite 数据库文件路径（仅 sqlite） */
      dbPath?: string;
      /** postgresql 模板（仅 pgsql） */
      template?: string;
      /** Redis 地址 host:port（必填） */
      redisAddr: string;
      /** Redis 密码（可空） */
      redisPassword: string;
      /** Redis 库号（默认 0） */
      redisDB: number;
    }

    /** /init/db/ping 请求体（数据库连接测试，不建库不落盘） */
    interface PingDBForm {
      dbType: DBType;
      host?: string;
      port?: string;
      userName?: string;
      password?: string;
      dbName: string;
      dbPath?: string;
      template?: string;
    }

    /** /init/redis/ping 请求体（Redis 连接测试） */
    interface PingRedisForm {
      addr: string;
      password?: string;
      db: number;
    }
  }
}
