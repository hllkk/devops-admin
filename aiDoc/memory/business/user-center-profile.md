# 个人中心:基本资料 + 头像上传后端接口

> 2026-07-22。前端 `views/_builtin/user-center` 骨架(`user-avatar.vue` cropper→toBlob、profile 表单、改密 tab)与 `service/api/system/user.ts` 三个 API 契约本就齐备,后端只缺 `profile`/`avatar` 两接口(改密 `profile/updatePwd` 早已实现)。本次补齐后端,前端零改动。

## 现状核对

- `PUT /system/user/profile/updatePwd`(`ChangeMyPassword`,密码过期强制改密入口)早已落地,不动
- 头像/网盘共用 `utils/upload` 的 OSS 抽象(`NewOss` 按 `System.OssType` 选 provider);RustFS 是 S3 兼容,后续切存储只需改 `config.yaml` 的 `oss-type` + `aws-s3`/`minio` endpoint,代码零改动——`config/oss_aws.go` 已有 `Endpoint`+`S3ForcePathStyle`+`DisableSSL` 三件套接自建 S3
- 全后端无人对 local url 补前缀(`GetUserInfo` 直接透传 `SysUser.Avatar`),avatar 存 `UploadFile` 返回的 url 与 media 一致

## 后端改动(4 文件)

- `model/system/request/sys_user.go`:新增 `UpdateMyProfileParams{nickName,email,phonenumber,sex}`,对齐前端 `UserProfileOperateParams`
- `service/system/sys_user_manage.go`:`UpdateMyProfile`(只写 4 字段 + `update_by`)、`UpdateMyAvatar`(`upload.NewOss().UploadFile` → 写回 `Avatar`,仅放行 jpg/jpeg/png/gif/webp)
- `api/v1/system/sys_user_manage.go`:`UpdateMyProfile`/`UpdateMyAvatar` handler + Swagger
- `router/system/sys_user.go`:注册 `PUT profile`、`POST profile/avatar`(与 `profile/updatePwd` 同组)

## 契约要点

- avatar 前端 FormData 字段名 `avatarfile`(`user-avatar.vue` cropper→`canvas.toBlob`→`append('avatarfile', blob)`);后端 `c.FormFile("avatarfile")`
- `isEncrypt:true` header 后端无解密逻辑,与 `ChangeMyPassword` 一致(明文 JSON、bcrypt 存储),本次未引入加密层
- 头像 url 存 `UploadFile` 返回值原样(local = `uploads/file/xxx` 相对路径),不补前缀,保持与 media 一致

## 已知点:local 模式头像刷新后可能 404

local `UploadFile` 返回相对 url `uploads/file/xxx`(无前导 `/`),上传瞬间前端用本地 blob 显示 OK;**刷新后** `userInfo.avatar` 相对当前页解析会 404。与现有 media 同属全站 local URL 策略问题,本次未单独特判。三解:A 切 rustfs/oss(绝对 URL,用户后续本要上,自然解);B nginx 反代 `/uploads/file`→`StaticFS`(`router.go:77` 已托管)+前端补 baseURL;C 后端统一给 local 补前导 `/`(会动 media 一致性)。用户倾向 A,暂不动。

## 验证

`go build ./...` + `go vet` + `gofmt -l` 全通过;前端零改动。

## 关联

- 用户管理基座见 [[user-management]](其「profile/updatePwd/avatar 待续」由本文件承接 profile/avatar;updatePwd 已于彼处落地)
- 存储抽象见 `utils/upload/`(OSS 接口 + `NewOss` 工厂);网盘规划见 `aiDoc/modules/business-modules.md`(未启动,RustFS)
